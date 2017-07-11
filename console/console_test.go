package console

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Console_Printf(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	New(buffer).Printf("hi")
	assert.Equal(t, "hi", buffer.String())
}

func Test_Console_WarnPrintf(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	New(buffer).Warn().Printf("hi")
	assert.Equal(t, "\033[1;33mhi\033[00m", buffer.String())
}

func Test_Console_SuccessPrintf(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	New(buffer).Success().Printf("hi")
	assert.Equal(t, "\033[0;32mhi\033[00m", buffer.String())
}

func Test_Console_NotifyPrintf(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	New(buffer).Notify().Printf("hi")
	assert.Equal(t, "\033[0;34mhi\033[00m", buffer.String())
}

func Test_Console_ErrorPrintf(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	New(buffer).Error().Printf("hi")
	assert.Equal(t, "\033[0;31mhi\033[00m", buffer.String())
}
