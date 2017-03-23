package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base32"
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

// GeneratePassword creates a random passoword  with len bytes of data
func GeneratePassword(len int) (string, error) {
	buffer := make([]byte, len)
	_, err := rand.Read(buffer)
	if err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(buffer), nil
}

// TryLoadKeys tries to load keys from keyPath. If no key exists generate it and save it
func TryLoadKeys(keyPath string) (ssh.Signer, error) {
	generateKey := func() (ssh.Signer, error) {
		private, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}

		// Try and save the key to a file
		bytes := x509.MarshalPKCS1PrivateKey(private)

		err = ioutil.WriteFile(keyPath, bytes, 0400)
		if err != nil {
			return nil, err
		}
		return ssh.NewSignerFromKey(private)
	}

	// Load the key from a file, create a new on on failure
	der, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return generateKey()
	}
	private, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return generateKey()
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		return generateKey()
	}
	return signer, nil
}

// GetFingerPrint takes returns the keys MD5 fingerprint of the signer
func GetFingerPrint(signer ssh.Signer) string {
	return ssh.FingerprintLegacyMD5(signer.PublicKey())
}

// CreatePasswordCallback creates a function for authenticating via password
func CreatePasswordCallback(masterPassword string) func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		if masterPassword == string(password) {
			return &ssh.Permissions{}, nil
		}
		return nil, fmt.Errorf("Invalid password")
	}
}
