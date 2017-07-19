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
	"github.com/efarrer/gmash/console"
	"github.com/efarrer/gmash/ip"
	"github.com/efarrer/gmash/ngrok"
	"github.com/efarrer/gmash/sshd"
	"github.com/efarrer/gmash/version"

	"golang.org/x/crypto/ssh"
)

func main() {
	console := console.New(os.Stdout)
	logger := log.New(os.Stderr, "", 0)

	console.Printf("GMASH (Version: %s)\n", version.String)
	latest, err := version.GetLatestVersion()
	if err != nil {
		console.Warn().Printf("Unable to find the latest version of gmash (%s)\n\n", err)
	} else if latest != version.String {
		console.Warn().Printf("A newer version (%s) of gmash is available!\n\n", latest)
	}

	var local = flag.Bool("local", false, "Whether to only allow connections over the local network")

	flag.Parse()

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
	fpMD5, fpSHA256 := auth.GetFingerPrint(signer)

	ctx, cancel := context.WithCancel(context.Background())

	listener, err := sshd.SSHServer("0.0.0.0:", &sshConf, shellConf)
	if err != nil {
		logger.Fatalf("%s\n", err)
	}
	defer func() { _ = listener.Close() }()

	var pubIP string
	port := listener.Addr().(*net.TCPAddr).Port

	if !*local {
		resp := ngrok.Execute(ctx, port)
		if resp.Err != nil {
			switch resp.Err.Reason {
			case ngrok.MissingNgrok:
				console.Warn().Printf("\nCan't find ngrok. Please install ngrok and make sure it's in your path. (See: https://ngrok.com/download)\n")
			case ngrok.UnexecutableNgrok:
				console.Warn().Printf("\nNgrok was found, but it couldn't be executed.")
			case ngrok.MissingAuthToken:
				console.Warn().Printf("\nNgrok's auth token must be installed. See: https://dashboard.ngrok.com/get-started\n")
			default:
				console.Warn().Printf("\n%s\n", resp.Err.Err.Error())
			}
			console.Warn().Printf("Due to errors executing ngrok. SSH server will only be available over the local network.\n\n")

			// We'll just have to treat this as a local connection
			*local = true
		} else {
			pubIP = resp.Value.Host
			port = resp.Value.Port
		}
	}

	if *local {
		pubIP, err = ip.LinuxPublicIP()
		if err != nil {
			logger.Fatalf("%s\n", err)
		}
	}

	console.Printf("Started server with RSA key: ")
	console.Success().Printf("%s\n", fpMD5)
	console.Printf("Started server with RSA key: ")
	console.Success().Printf("%s\n", fpSHA256)
	console.Printf("\n")
	console.Printf("To connect type:\n")
	console.Notify().Printf("ssh -o UserKnownHostsFile=/dev/null %s -p %d\n\n", pubIP, port)
	console.Printf("password: ")
	console.Success().Printf("%s\n", masterPassword)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	select {
	case <-signalCh:
		cancel()
		fmt.Printf("Bubye\n")
	case <-ctx.Done():
	}
}
