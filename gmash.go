package main

import (
	"fmt"
	"gmash/auth"
	"gmash/ip"
	"gmash/sshd"
	"log"
	"net"
	"os"
	"os/user"
	"path"

	"golang.org/x/crypto/ssh"
)

func main() {
	// Get the user's home directory
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Unable to get user's home directory (%s)\n", err)
	}
	gmashDir := path.Join(usr.HomeDir, ".gmash")

	// Create the gmash dir
	err = os.MkdirAll(gmashDir, 0700)
	if err != nil {
		log.Fatalf("Unable to create %s (%s)\n", gmashDir, err)
	}

	// Generate a random user password for this session
	masterPassword, err := auth.GeneratePassword(10)
	if err != nil {
		log.Fatalf("Unable to generate password (%s)", err)
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
		log.Fatal(err)
	}
	sshConf.AddHostKey(signer)

	pubIP, err := ip.LinuxPublicIP()
	if err != nil {
		log.Fatalf("%s\n", err)
	}

	listener, err := sshd.SSHServer(pubIP+":", &sshConf, shellConf)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	defer func() { _ = listener.Close() }()

	fmt.Printf("Started server with RSA key: %s\n", auth.GetFingerPrint(signer))
	fmt.Printf("To connect type:\n")
	fmt.Printf("ssh -o UserKnownHostsFile=/dev/null %s -p %d\n", pubIP, listener.Addr().(*net.TCPAddr).Port)
	fmt.Printf("password %s\n", masterPassword)

	select {}
}
