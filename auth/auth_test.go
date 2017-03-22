package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenratePassword_IsRandom(t *testing.T) {
	pw0, err := GeneratePassword(10)
	assert.Nil(t, err)
	pw1, err := GeneratePassword(10)
	assert.Nil(t, err)
	assert.NotEqual(t, pw0, pw1)
}

func TestGenerateKeys_HappyPath(t *testing.T) {
	_, err := GenerateKeys()
	assert.Nil(t, err)
}

func TestGetFingerPrint_HappyPath(t *testing.T) {
	signer, _ := GenerateKeys()
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
