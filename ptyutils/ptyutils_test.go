package ptyutils

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/kr/pty"
	"github.com/stretchr/testify/assert"
)

func TestSetWindowSize_FailsWithANonPty(t *testing.T) {
	file, err := ioutil.TempFile("", "SetWindowSize")
	assert.NoError(t, err)
	defer func() { _ = os.Remove(file.Name()) }()

	err = SetWindowSize(file, 100, 100)
	assert.Error(t, err)
}

func TestSetWindowSize_WorksWithPty(t *testing.T) {
	_pty, tty, err := pty.Open()
	assert.NoError(t, err)
	defer func() { _ = _pty.Close() }()
	defer func() { _ = tty.Close() }()

	width := 100
	height := 50

	err = SetWindowSize(_pty, width, height)
	assert.NoError(t, err)

	rows, cols, err := pty.Getsize(_pty)
	assert.Equal(t, width, cols)
	assert.Equal(t, height, rows)
}
