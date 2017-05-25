package sshd

import (
	"bytes"
	"errors"
	"fmt"
	"gmash/auth"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/stretchr/testify/assert"
)

// readCloser is a fake thread-safe io.ReadCloser.
type readCloser struct {
	toRead     []byte
	readError  error
	closeError error
	lock       sync.Mutex
}

func newReadCloser(toRead []byte, readError, closeError error) *readCloser {
	return &readCloser{
		toRead:     toRead,
		readError:  readError,
		closeError: closeError,
	}
}

func (rc *readCloser) Read(p []byte) (n int, err error) {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	if rc.readError != nil {
		return 0, rc.readError
	}
	if len(rc.toRead) == 0 {
		return 0, io.EOF
	}
	val := copy(p, rc.toRead)
	rc.toRead = rc.toRead[val:]
	return val, nil
}

func (rc *readCloser) Close() error {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	return rc.closeError
}

// writeCloser is a thread-safe io.WriteCloser
type writeCloser struct {
	wrote      []byte
	closeError error
	lock       sync.Mutex
}

func newWriteCloser(closeError error) *writeCloser {
	return &writeCloser{
		closeError: closeError,
	}
}

func (wc *writeCloser) Write(data []byte) (int, error) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	wc.wrote = append(wc.wrote, data...)
	return len(data), nil
}

func (wc *writeCloser) Close() error {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	return wc.closeError
}

func (wc *writeCloser) Bytes() []byte {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	return wc.wrote
}

// fakeChannel is a thread-safe ssh.Channel
type fakeChannel struct {
	readCloser
	writeCloser
}

func newFakeChannel(toRead []byte, closeError error) *fakeChannel {
	return &fakeChannel{
		readCloser: readCloser{
			toRead:     toRead,
			closeError: closeError,
		},
		writeCloser: writeCloser{
			closeError: closeError,
		},
	}
}

func (fc *fakeChannel) Close() error {
	fc.readCloser.lock.Lock()
	defer fc.readCloser.lock.Unlock()
	fc.writeCloser.lock.Lock()
	defer fc.writeCloser.lock.Unlock()
	return fc.readCloser.closeError
}

func (fc *fakeChannel) CloseWrite() error {
	return nil
}

func (fc *fakeChannel) SendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	return true, nil
}

func (fc *fakeChannel) Stderr() io.ReadWriter {
	return nil
}

type mockShellConf struct {
	shell string
	err   error
}

func (sc *mockShellConf) Shell() string {
	return sc.shell
}

func (sc *mockShellConf) ErrorHandler(err error) {
	sc.err = err
}

func newShellConf() *mockShellConf {
	return &mockShellConf{shell: "/bin/bash"}
}

func TestHandlePtyRequest_WithInvalidPtyPayloadReturnsError(t *testing.T) {
	channel := newFakeChannel([]byte{}, nil)

	err := handlePtyRequest("/bin/bash", channel, &ssh.Request{})

	assert.Error(t, err)
}

var ptyPayload = []byte{
	0x0, 0x0, 0x0, 0x1, 0x3b, // tty type
	0x0, 0x0, 0x0, 0x0A, // width chars
	0x0, 0x0, 0x0, 0xA0, // height columns
	0x0, 0x0, 0x0, 0x0, // width pixels
	0x0, 0x0, 0x0, 0x0, // terminal modes
}

func TestHandlePtyRequest_WithInvalidShellReturnsError(t *testing.T) {
	channel := newFakeChannel([]byte{}, nil)

	err := handlePtyRequest("/", channel, &ssh.Request{
		Payload: ptyPayload,
	})

	assert.Error(t, err)
}

func createTestBinary() (string, func(), error) {
	// Create a test binary to run
	dir := "testing"
	source := path.Join(dir, "./testing.go")
	bin := path.Join(dir, "./testing")
	text := `package main
		import "fmt"
		func main() {
			fmt.Printf("hi\n")
		}
		`
	os.RemoveAll(dir)
	err := os.Mkdir(dir, 0777)
	if err != nil {
		return "", func() {}, err
	}
	err = ioutil.WriteFile(source, []byte(text), 0444)
	if err != nil {
		return "", func() {}, err
	}
	err = exec.Command("go", "build", "-o", bin, source).Run()
	if err != nil {
		return "", func() {}, err
	}
	return bin, func() {
		os.RemoveAll(dir)
	}, nil
}

func TestHandlePtyRequest_HappyPath(t *testing.T) {
	bin, closer, err := createTestBinary()
	assert.NoError(t, err)
	defer closer()

	channel := newFakeChannel([]byte{}, nil)

	err = handlePtyRequest(bin, channel, &ssh.Request{
		Payload: ptyPayload,
	})

	assert.NoError(t, err)
}

func startReqChan(req *ssh.Request) chan *ssh.Request {
	reqCh := make(chan *ssh.Request)
	go func() {
		reqCh <- req
		close(reqCh)
	}()
	return reqCh
}

func TestHandleSshRequests_HandlesEmptyRequests(t *testing.T) {
	sc := newShellConf()
	channel := newFakeChannel([]byte{}, nil)
	reqCh := startReqChan(nil)
	<-reqCh // Swallow the req

	handleSSHRequests(channel, reqCh, sc)

	assert.NoError(t, sc.err)
}

func TestHandleSshRequests_SkipsBogusRequesets(t *testing.T) {
	sc := newShellConf()
	channel := newFakeChannel([]byte{}, nil)
	reqCh := startReqChan(&ssh.Request{Type: "bogus", WantReply: false})

	handleSSHRequests(channel, reqCh, sc)

	assert.NoError(t, sc.err)
}

func TestHandleSshRequests_HandlesPtyRequestErrors(t *testing.T) {
	sc := newShellConf()
	channel := newFakeChannel([]byte{}, nil)
	reqCh := startReqChan(&ssh.Request{Type: "pty-req"})
	// override handlePtyRequest then restore it later
	handlePtyRequest = func(string, ssh.Channel, *ssh.Request) error {
		return errors.New("some error")
	}
	defer setupFunctionPointers()

	handleSSHRequests(channel, reqCh, sc)

	assert.Error(t, sc.err)
}

type fakeNewChannel struct {
	channelType string
	acceptError error
}

func (fc fakeNewChannel) ChannelType() string {
	return fc.channelType
}

func (fc fakeNewChannel) Accept() (ssh.Channel, <-chan *ssh.Request, error) {
	return nil, nil, fc.acceptError
}

func (fc fakeNewChannel) Reject(reason ssh.RejectionReason, message string) error {
	return nil
}

func (fc fakeNewChannel) ExtraData() []byte {
	return nil
}

func startNewChannelChannel(fcs []fakeNewChannel) <-chan ssh.NewChannel {
	ch := make(chan ssh.NewChannel)
	go func() {
		for i := 0; i != len(fcs); i++ {
			ch <- fcs[i]
		}
		close(ch)
	}()
	return ch
}

func TestProcessSshChannels_IgnoresNonSessionChannelTypes(t *testing.T) {
	sc := newShellConf()
	newChannelChan := startNewChannelChannel([]fakeNewChannel{{channelType: "bogus", acceptError: nil}})

	processSSHChannels(newChannelChan, sc)

	assert.Error(t, sc.err)
}

func TestProcessSshChannels_HandlesErrorsAcceptingChannelRequest(t *testing.T) {
	sc := newShellConf()
	newChannelChan := startNewChannelChannel([]fakeNewChannel{{channelType: "session", acceptError: errors.New("")}})

	processSSHChannels(newChannelChan, sc)

	assert.Error(t, sc.err)
}

func TestProcessSshChannels_HandlesSessionRequests(t *testing.T) {
	sc := newShellConf()
	newChannelChan := startNewChannelChannel([]fakeNewChannel{{channelType: "session", acceptError: nil}})

	// override handleSSHRequests then restore it later
	ch := make(chan struct{})
	handleSSHRequests = func(ssh.Channel, <-chan *ssh.Request, ShellConf) {
		ch <- struct{}{}
	}
	defer setupFunctionPointers()

	processSSHChannels(newChannelChan, sc)
	// This will block unless handleSSHRequests is called above
	<-ch

	assert.NoError(t, sc.err)
}

func TestProcessSSHConnection_HandlesHandshakeErrors(t *testing.T) {
	sshConf := &ssh.ServerConfig{}
	sc := newShellConf()
	cli, srv := net.Pipe()
	_ = cli.Close()
	_ = srv.Close()

	processSSHConnection(srv, sshConf, sc)
}

func TestProcessSSHConnection_ProcessesChannels(t *testing.T) {
	sshConf := &ssh.ServerConfig{}
	sc := newShellConf()
	cli, srv := net.Pipe()
	_ = cli.Close()
	_ = srv.Close()

	newServerConn = func(c net.Conn, config *ssh.ServerConfig) (*ssh.ServerConn, <-chan ssh.NewChannel, <-chan *ssh.Request, error) {
		return nil, startNewChannelChannel([]fakeNewChannel{}), startReqChan(nil), nil
	}
	processCalled := false
	processSSHChannels = func(sshChan <-chan ssh.NewChannel, shellConf ShellConf) {
		processCalled = true
	}
	discardCalled := make(chan bool)
	discardRequests = func(<-chan *ssh.Request) {
		discardCalled <- true
	}
	defer setupFunctionPointers()

	processSSHConnection(srv, sshConf, sc)

	assert.True(t, processCalled)
	assert.True(t, <-discardCalled)
}

func TestSSHServer_ReturnsErrorWithBadAddress(t *testing.T) {
	sshConf := &ssh.ServerConfig{}
	sc := newShellConf()
	_, err := SSHServer("bogus", sshConf, sc)
	assert.Error(t, err)
}

func TestSSHServer_ProcessesConnections(t *testing.T) {
	sshConf := &ssh.ServerConfig{}
	sc := newShellConf()
	funcCalled := make(chan bool)
	processSSHConnection = func(net.Conn, *ssh.ServerConfig, ShellConf) {
		funcCalled <- true
	}
	defer setupFunctionPointers()
	listener, err := SSHServer("127.0.0.1:", sshConf, sc)
	assert.NoError(t, err)
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", listener.Addr().(*net.TCPAddr).Port))
	assert.NoError(t, err)
	listener.Close()
	conn.Close()
	assert.True(t, <-funcCalled)
}

func createTestServer(shell string) (int, func(), error) {
	sshConf := ssh.ServerConfig{NoClientAuth: true}
	shellConf := DefaultShellConf(shell, func(err error) {})
	signer, err := auth.TryLoadKeys("/dev/null")
	if err != nil {
		return 0, func() {}, err
	}
	sshConf.AddHostKey(signer)
	listener, err := SSHServer("127.0.0.1:", &sshConf, shellConf)
	if err != nil {
		return 0, func() {}, err
	}
	return listener.Addr().(*net.TCPAddr).Port, func() {
		listener.Close()
	}, nil
}

func TestIntegration_ServerDisconnects(t *testing.T) {
	bin, closer, err := createTestBinary()
	assert.NoError(t, err)
	defer closer()

	port, closer, err := createTestServer(bin)
	assert.NoError(t, err)
	defer closer()

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := exec.Command("/usr/bin/ssh", "-t", "-t", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "-p", strconv.Itoa(port), "localhost").Run()
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestIntegration_ClientDisconnects(t *testing.T) {
	bin := "/bin/bash"

	port, closer, err := createTestServer(bin)
	assert.NoError(t, err)
	defer closer()

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command("/usr/bin/ssh", "-t", "-t", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "-p", strconv.Itoa(port), "localhost")
			cmd.Stdin = bytes.NewBuffer([]byte("exit\n"))
			cmd.Start()
			err := cmd.Wait()
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func BenchmarkIntegration_ServerDisconnects(b *testing.B) {
	bin, closer, err := createTestBinary()
	if err != nil {
		b.Fatalf("%s\n", err)
	}
	defer closer()

	port, closer, err := createTestServer(bin)
	if err != nil {
		b.Fatalf("%s\n", err)
	}
	defer closer()

	var wg sync.WaitGroup
	for i := 0; i < 1025; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := exec.Command("/usr/bin/ssh", "-t", "-t", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "-p", strconv.Itoa(port), "localhost").Run()
			if err != nil {
				b.Fatalf("%s\n", err)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkIntegration_ClientDisconnects(b *testing.B) {
	bin := "/bin/bash"

	port, closer, err := createTestServer(bin)
	if err != nil {
		b.Fatalf("%s\n", err)
	}
	defer closer()

	var wg sync.WaitGroup
	for i := 0; i < 125; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command("/usr/bin/ssh", "-t", "-t", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "-p", strconv.Itoa(port), "localhost")
			cmd.Stdin = bytes.NewBuffer([]byte("exit\n"))
			cmd.Start()
			err := cmd.Wait()
			if err != nil {
				b.Fatalf("%s\n", err)
			}
		}()
	}
	wg.Wait()
}
