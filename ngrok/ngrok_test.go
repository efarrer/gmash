package ngrok

import (
	"context"
	"os"
	"os/exec"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecute_MissingBinary(t *testing.T) {
	command := "sadflkasdjksfadjfds"
	resp := execute(context.Background(), 100, command)
	assert.NotNil(t, resp.Err)
	assert.Equal(t, resp.Err.Reason, MissingNgrok)
}

func TestExecute_NotExecutable(t *testing.T) {
	resp := execute(context.Background(), 100, "/dev/null")
	assert.NotNil(t, resp.Err)
	assert.Equal(t, resp.Err.Reason, UnexecutableNgrok)
}

func buildFakeNgrok(t *testing.T) (string, func()) {
	dir := "fakengrok"
	exe := "fakengrok"
	path := path.Join(dir, exe)
	// Compile our fake ngrok program
	cmd := exec.Command("go", "build")
	cmd.Dir = dir
	err := cmd.Run()
	assert.NoError(t, err)
	return path, func() { _ = os.Remove(path) }
}

func TestExecute_MissingAuthToken(t *testing.T) {
	path, closer := buildFakeNgrok(t)
	defer closer()

	tests := []struct {
		delayMS    int
		characters int
	}{
		{0, 20000},
		{250, 20000},
		{10, 10},
	}

	err := os.Setenv("TYPE", "NOAUTH")
	assert.NoError(t, err)
	for _, test := range tests {
		err := os.Setenv("DELAY_MS", strconv.Itoa(test.delayMS))
		assert.NoError(t, err)
		err = os.Setenv("CHARACTERS", strconv.Itoa(test.characters))
		assert.NoError(t, err)

		resp := execute(context.Background(), 100, path)
		assert.NotNil(t, resp.Err)
		assert.Equal(t, resp.Err.Reason, MissingAuthToken)
	}
}

func TestExecute_WithCanceledContext(t *testing.T) {
	path, closer := buildFakeNgrok(t)
	defer closer()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resp := execute(ctx, 100, path)
	assert.NotNil(t, resp.Err)
	assert.Equal(t, resp.Err.Reason, Canceled)
}

func TestExecute_WithDelayedCanceledContext(t *testing.T) {
	path, closer := buildFakeNgrok(t)
	defer closer()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	err := os.Setenv("DELAY_MS", "250")
	assert.NoError(t, err)
	err = os.Setenv("CHARACTERS", "100")
	assert.NoError(t, err)
	resp := execute(ctx, 100, path)
	assert.NotNil(t, resp.Err)
	assert.Equal(t, resp.Err.Reason, Canceled)
}

func TestExecute_NgrokFails(t *testing.T) {
	path, closer := buildFakeNgrok(t)
	defer closer()

	tests := []struct {
		delayMS    int
		characters int
	}{
		{0, 20000},
		{250, 20000},
		{10, 10},
	}

	err := os.Setenv("TYPE", "bogus")
	assert.NoError(t, err)
	for _, test := range tests {
		err := os.Setenv("DELAY_MS", strconv.Itoa(test.delayMS))
		assert.NoError(t, err)
		err = os.Setenv("CHARACTERS", strconv.Itoa(test.characters))
		assert.NoError(t, err)
		err = os.Setenv("HANG_HOURS", "0")
		assert.NoError(t, err)

		resp := execute(context.Background(), 100, path)
		assert.NotNil(t, resp.Err)
		assert.Equal(t, resp.Err.Reason, CantReadFromPty)
	}
}

func TestExecute_NgrokInvalidURL(t *testing.T) {
	path, closer := buildFakeNgrok(t)
	defer closer()

	err := os.Setenv("TYPE", "Forwarding tcp://[fe80::%31] ")
	assert.NoError(t, err)

	resp := execute(context.Background(), 100, path)
	assert.NotNil(t, resp.Err)
	assert.Equal(t, resp.Err.Reason, URLParsingError)
}

func TestExecute_NgrokInvalidPort(t *testing.T) {
	path, closer := buildFakeNgrok(t)
	defer closer()

	err := os.Setenv("TYPE", "Forwarding tcp://foo:abcd ")
	assert.NoError(t, err)

	resp := execute(context.Background(), 100, path)
	assert.NotNil(t, resp.Err)
	assert.Equal(t, resp.Err.Reason, PortParsingError)
}

func TestExecute_RetrievesNgroksUrl(t *testing.T) {
	path, closer := buildFakeNgrok(t)
	defer closer()

	tests := []struct {
		delayMS    int
		characters int
	}{
		{0, 20000},
		{250, 20000},
		{10, 10},
	}

	err := os.Setenv("TYPE", "VALID")
	assert.NoError(t, err)
	for _, test := range tests {
		err := os.Setenv("DELAY_MS", strconv.Itoa(test.delayMS))
		assert.NoError(t, err)
		err = os.Setenv("CHARACTERS", strconv.Itoa(test.characters))
		assert.NoError(t, err)

		resp := execute(context.Background(), 100, path)
		assert.Nil(t, resp.Err)
		assert.NotNil(t, resp.Value)
		assert.Equal(t, resp.Value.Host, "0.tcp.ngrok.io")
		assert.Equal(t, resp.Value.Port, 15120)
	}
}
