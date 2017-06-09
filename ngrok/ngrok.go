package ngrok

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/efarrer/gmash/ptyutils"
	"github.com/kr/pty"
)

// Reason is the reason why executing ngrok failed
type Reason int

const (
	// 0 is unused to help find issues with errors that have a default value
	_ Reason = iota
	// MissingNgrok indicates ngrok executable can't be found
	MissingNgrok Reason = iota
	// UnexecutableNgrok indicates ngrok can't be executed
	UnexecutableNgrok Reason = iota
	// MissingAuthToken indicates ngrok must be authed
	MissingAuthToken Reason = iota
	// Canceled indicates that the user canceled the execution
	Canceled Reason = iota
	// CantReadFromPty indicates that there was a problem reading the stdout from ngrok
	CantReadFromPty Reason = iota
	// PortParsingError indicates that there was a problem parsing the forwarding url's port from ngrok
	PortParsingError Reason = iota
	// URLParsingError indicates that there was a problem parsing the forwarding url from ngrok
	URLParsingError Reason = iota
	// CantSetPtyWindowSize indicates that there was a problem setting the pty's window size
	CantSetPtyWindowSize Reason = iota
)

// ExecutionError is an error type returned by Execute
type ExecutionError struct {
	Reason Reason
	Err    error
}

// Error returns the error string
func (r *ExecutionError) Error() string {
	return fmt.Sprintf("Error(%d) %s", r.Reason, r.Err)
}

// Value is the Host and Port found by executing ngrok
type Value struct {
	Host string
	Port int
}

// A Response contains either an error from executing ngrok or the Value
type Response struct {
	Err   *ExecutionError
	Value *Value
}

// String returns a human friendly representation of a Response
func (rs Response) String() string {
	res := "ngrok.Response"
	if rs.Err != nil {
		res += fmt.Sprintf("%d %s", rs.Err.Reason, rs.Err.Err)
	}
	if rs.Value != nil {
		res += fmt.Sprintf("%s %s", rs.Value.Host, rs.Value.Port)
	}
	return res
}

func newErrorResponse(reason Reason, err error) Response {
	return Response{
		Err: &ExecutionError{
			Reason: reason,
			Err:    err,
		},
		Value: nil,
	}
}

// Execute executes ngrok forwarding to the given port
func Execute(ctx context.Context, port int) Response {
	return execute(ctx, port, "ngrok")
}

func execute(ctx context.Context, port int, bin string) Response {
	cmd := exec.CommandContext(ctx, bin, "tcp", strconv.FormatInt(int64(port), 10))
	_pty, err := pty.Start(cmd)
	if err != nil {
		var reason Reason
		switch err.(type) {
		case *exec.Error:
			reason = MissingNgrok
		case *os.PathError:
			reason = UnexecutableNgrok
		default:
			reason = Canceled
		}

		return newErrorResponse(reason, err)
	}

	err = ptyutils.SetWindowSize(_pty, 100, 100)
	if err != nil {
		return newErrorResponse(CantSetPtyWindowSize, err)
	}

	output := ""
	bytes := make([]byte, 1024)
	for {
		count, err := _pty.Read(bytes)
		if err != nil {
			if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
				return newErrorResponse(
					Canceled,
					errors.New("ngrok was canceled"),
				)
			}
			return newErrorResponse(CantReadFromPty, err)
		}
		output += string(bytes[0:count])

		if strings.Contains(output, "ERR_NGROK_302") {
			return newErrorResponse(
				MissingAuthToken,
				errors.New("Please signup at https://ngrok.com/signup or make sure your athtoken is installed https://dashboard.ngrok.com"),
			)
		}
		rx := regexp.MustCompile("Forwarding[ ]+(tcp://[^ ]+)[ ].*")

		match := rx.FindStringSubmatch(output)
		if len(match) == 2 {
			ngrokurl, err := url.Parse(match[1])
			if err != nil {
				return newErrorResponse(
					URLParsingError,
					errors.New("Unable to parse ngrok's forwarding url"),
				)
			}
			iport, err := strconv.Atoi(ngrokurl.Port())
			if err != nil {
				return newErrorResponse(
					PortParsingError,
					errors.New("Unable to parse ngrok's port"),
				)
			}

			return Response{
				Err:   nil,
				Value: &Value{Host: ngrokurl.Hostname(), Port: iport},
			}
		}
	}
}
