package main

import (
	"fmt"
	"gmash/auth"
	"gmash/sshd"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
)

func main() {
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
	signer, err := auth.GenerateKeys()
	if err != nil {
		log.Fatal(err)
	}
	sshConf.AddHostKey(signer)

	listener, err := sshd.SSHServer("127.0.0.1:", &sshConf, shellConf)
	if err != nil {
		log.Fatalf("%s\n", err)
	}

	fmt.Printf("Started server with RSA key: %s\n", auth.GetFingerPrint(signer))
	fmt.Printf("To connect type:\n")
	fmt.Printf("ssh -o UserKnownHostsFile=/dev/null localhost -p %d\n", listener.Addr().(*net.TCPAddr).Port)
	fmt.Printf("password %s\n", masterPassword)
	defer func() { _ = listener.Close() }()
	select {}
}
