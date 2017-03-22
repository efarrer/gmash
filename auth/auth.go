package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base32"
	"fmt"

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

// GenerateKeys generates ssh server keys
func GenerateKeys() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.NewSignerFromKey(key)

	return signer, err
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
