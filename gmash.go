package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path"

	"github.com/efarrer/gmash/auth"
	"github.com/efarrer/gmash/ip"
	"github.com/efarrer/gmash/ngrok"
	"github.com/efarrer/gmash/sshd"

	"golang.org/x/crypto/ssh"
)

func main() {
	logger := log.New(os.Stderr, "", 0)

	var local = flag.Bool("local", false, "Whether to only allow connections over the local network")
	var global = flag.Bool("global", false, "Whether to allow connections from anywhere")

	flag.Parse()

	if !*local && !*global {
		logger.Fatal("You must specify either the -local or -global arguments")
	}

	// Get the user's home directory
	usr, err := user.Current()
	if err != nil {
		logger.Fatalf("Unable to get user's home directory (%s)\n", err)
	}
	gmashDir := path.Join(usr.HomeDir, ".gmash")

	// Create the gmash dir
	err = os.MkdirAll(gmashDir, 0700)
	if err != nil {
		logger.Fatalf("Unable to create %s (%s)\n", gmashDir, err)
	}

	// Generate a random user password for this session
	masterPassword, err := auth.GeneratePassword(10)
	if err != nil {
		logger.Fatalf("Unable to generate password (%s)", err)
	}

	// Construct the ssh configuration with password authentication
	sshConf := ssh.ServerConfig{
		PasswordCallback: auth.CreatePasswordCallback(masterPassword),
	}
	shellConf := sshd.DefaultShellConf(
		"/bin/bash",
		func(err error) { fmt.Printf("%s\n", err) },
	)

	// Generate server ssh keys
	signer, err := auth.TryLoadKeys(path.Join(gmashDir, "key"))
	if err != nil {
		logger.Fatal(err)
	}
	sshConf.AddHostKey(signer)

	pubIP, err := ip.LinuxPublicIP()
	if err != nil {
		logger.Fatalf("%s\n", err)
	}

	if *global {
		pubIP = "0.0.0.0"
	}

	listener, err := sshd.SSHServer(pubIP+":", &sshConf, shellConf)
	if err != nil {
		logger.Fatalf("%s\n", err)
	}
	defer func() { _ = listener.Close() }()

	ctx, cancel := context.WithCancel(context.Background())

	port := listener.Addr().(*net.TCPAddr).Port
	if *global {
		resp := ngrok.Execute(ctx, port)
		if resp.Err != nil {
			switch resp.Err.Reason {
			case ngrok.MissingNgrok:
				logger.Fatalf("Can't find ngrok. Please install ngrok and make sure it's in your path. https://ngrok.com/download")
			case ngrok.UnexecutableNgrok:
				logger.Fatalf("Ngrok was found, but it couldn't be executed.")
			case ngrok.MissingAuthToken:
				logger.Fatalf("Ngrok's auth token must be installed. See: https://dashboard.ngrok.com/get-started")
			default:
				logger.Fatalf(resp.Err.Err.Error())
			}
		}
		pubIP = resp.Value.Host
		port = resp.Value.Port
	}

	fpMD5, fpSHA256 := auth.GetFingerPrint(signer)
	fmt.Printf("Started server with RSA key: %s\n", fpMD5)
	fmt.Printf("Started server with RSA key: %s\n", fpSHA256)
	fmt.Println("")
	fmt.Printf("To connect type:\n")
	fmt.Printf("ssh -o UserKnownHostsFile=/dev/null %s -p %d\n", pubIP, port)
	fmt.Printf("password: %s\n", masterPassword)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	select {
	case <-signalCh:
		cancel()
		fmt.Printf("Bubye\n")
	case <-ctx.Done():
	}
}
