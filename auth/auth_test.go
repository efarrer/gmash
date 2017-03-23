package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/stretchr/testify/assert"
)

func TestGenratePassword_IsRandom(t *testing.T) {
	pw0, err := GeneratePassword(10)
	assert.Nil(t, err)
	pw1, err := GeneratePassword(10)
	assert.Nil(t, err)
	assert.NotEqual(t, pw0, pw1)
}

func TestTryLoadKeys_GeneratesKeyIfKeyDoesntExist(t *testing.T) {
	dir, err := ioutil.TempDir("", "key")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(dir) }()

	key := path.Join(dir, "key")
	signer, err := TryLoadKeys(key)

	assert.NotNil(t, signer)
	assert.NoError(t, err)

	// The file should exist
	_, err = os.Stat(key)
	assert.NoError(t, err)
}

func TestTryLoadKeys_LoadsKeyIfItExists(t *testing.T) {
	dir, err := ioutil.TempDir("", "key")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(dir) }()

	key := path.Join(dir, "key")
	signer, err := TryLoadKeys(key)
	assert.NotNil(t, signer)
	assert.NoError(t, err)

	signer2, err := TryLoadKeys(key)
	assert.NotNil(t, signer2)
	assert.NoError(t, err)

	assert.Equal(t, signer, signer2)
}

func TestTryLoadKeys_ReturnsErrorIfCantSaveFile(t *testing.T) {
	signer, err := TryLoadKeys("")
	assert.Nil(t, signer)
	assert.Error(t, err)
}

func TestGetFingerPrint_HappyPath(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(key)
	assert.NoError(t, err)

	fp := GetFingerPrint(signer)
	assert.Equal(t, len(fp), 47)
}

func TestCreatePasswordCallback_MatchingPassword(t *testing.T) {
	_, err := CreatePasswordCallback("hi")(nil, []byte("hi"))
	assert.Nil(t, err)
}

func TestCreatePasswordCallback_BadPassword(t *testing.T) {
	_, err := CreatePasswordCallback("hi")(nil, []byte("bad"))
	assert.NotNil(t, err)
}
