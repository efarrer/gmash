package sshd

import (
	"fmt"
	"io"
	"net"
	"os/exec"

	"github.com/efarrer/gmash/payload"

	"github.com/kr/pty"

	"golang.org/x/crypto/ssh"
)

// Using local function vars to facilitate mocks for tests
var handlePtyRequest func(string, ssh.Channel, *ssh.Request) error
var handleSSHRequests func(channel ssh.Channel, reqsCh <-chan *ssh.Request, shellConf ShellConf)
var processSSHChannels func(sshChan <-chan ssh.NewChannel, shellConf ShellConf)
var newServerConn func(net.Conn, *ssh.ServerConfig) (*ssh.ServerConn, <-chan ssh.NewChannel, <-chan *ssh.Request, error)
var discardRequests func(in <-chan *ssh.Request)
var processSSHConnection func(conn net.Conn, sshConf *ssh.ServerConfig, shellConf ShellConf)

func init() {
	setupFunctionPointers()
}

func setupFunctionPointers() {
	handlePtyRequest = _handlePtyRequest
	handleSSHRequests = _handleSSHRequests
	processSSHChannels = _processSSHChannels
	newServerConn = _newServerConn
	discardRequests = _discardRequests
	processSSHConnection = _processSSHConnection
}

// A ShellConf has common configuration for a ssh shell
type ShellConf interface {
	Shell() string
	ErrorHandler(error)
}

type shellConf struct {
	shell        string
	errorHandler func(error)
}

// DefaultShellConf creates the standard ShellConf
func DefaultShellConf(shell string, errorHandler func(error)) ShellConf {
	return &shellConf{
		shell:        shell,
		errorHandler: errorHandler,
	}
}

func (sc *shellConf) Shell() string {
	return sc.shell
}

func (sc *shellConf) ErrorHandler(err error) {
	sc.errorHandler(err)
}

func _handlePtyRequest(shell string, channel ssh.Channel, req *ssh.Request) error {
	ptyReq, err := payload.ParsePtyReq(req.Payload)
	if err != nil {
		return fmt.Errorf("Unable to parse pty request (%s)", err)
	}
	_ = ptyReq
	// TODO set the PTY size from ptyReq
	ptyFile, err := pty.Start(exec.Command(shell))
	if err != nil {
		return fmt.Errorf("Unable to create pty request (%s)", err)
	}

	doneCh := make(chan struct{})
	// Note that channel is a ReadWriter to handling the requests stdin and
	// stdout. Stderr is with channel.Stderr()
	go func() {
		_, _ = io.Copy(channel, ptyFile)
		doneCh <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(ptyFile, channel)
		doneCh <- struct{}{}
	}()

	go func() {
		<-doneCh
		// TODO get the actual exit status instead of just using 0
		channel.SendRequest("exit-status", false, ssh.Marshal(&struct{ ExitStatus uint32 }{0}))
		channel.Close()
		ptyFile.Close()
		<-doneCh
	}()

	return nil
}

func _handleSSHRequests(channel ssh.Channel, reqsCh <-chan *ssh.Request, shellConf ShellConf) {
	for req := range reqsCh {
		var err error
		switch req.Type {
		case "pty-req":
			err = handlePtyRequest(shellConf.Shell(), channel, req)
			if err != nil {
				shellConf.ErrorHandler(err)
				continue
			}
		}
		if req.WantReply {
			err = req.Reply(err == nil, nil)
			if err != nil {
				shellConf.ErrorHandler(err)
			}
		}
	}
}

func _processSSHChannels(sshChan <-chan ssh.NewChannel, shellConf ShellConf) {
	for newChannel := range sshChan {
		if newChannel.ChannelType() != "session" {
			shellConf.ErrorHandler(fmt.Errorf("unsupported channel type : %v", newChannel.ChannelType()))
			err := newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			if err != nil {
				shellConf.ErrorHandler(err)
			}
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			shellConf.ErrorHandler(fmt.Errorf("could not accept channel: %v", err))
			continue
		}

		go handleSSHRequests(channel, requests, shellConf)
	}
}

func _newServerConn(c net.Conn, config *ssh.ServerConfig) (*ssh.ServerConn, <-chan ssh.NewChannel, <-chan *ssh.Request, error) {
	return ssh.NewServerConn(c, config)
}

func _discardRequests(in <-chan *ssh.Request) {
	ssh.DiscardRequests(in)
}

func _processSSHConnection(conn net.Conn, sshConf *ssh.ServerConfig, shellConf ShellConf) {
	defer func() { _ = conn.Close() }()

	// Establish the ssh connection
	_, sshChan, sshRequest, err := newServerConn(conn, sshConf)
	if err != nil {
		shellConf.ErrorHandler(fmt.Errorf("failed to establish ssh connection (%s)", err))
		return
	}

	// Yea were not going to handle any requests (port/X11 forwarding etc. at this time)
	go discardRequests(sshRequest)

	processSSHChannels(sshChan, shellConf)
}

// SSHServer starts an ssh server on the given address
func SSHServer(addr string, sshConf *ssh.ServerConfig, shellConf ShellConf) (net.Listener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s (%s)", addr, err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				shellConf.ErrorHandler(fmt.Errorf("failed to accept TCP connection (%v)", err))
				return
			}

			go processSSHConnection(conn, sshConf, shellConf)
		}
	}()
	return listener, nil
}
