package console

import (
	"fmt"
	"io"
)

const (
	yellow        = "\033[1;33m"
	green         = "\033[0;32m"
	red           = "\033[0;31m"
	blue          = "\033[0;34m"
	default_color = "\033[00m"
)

// A Printer writes stuff to a console
type Printer interface {
	Printf(format string, a ...interface{}) (n int, err error)
}

// A Console allows writing to the console
type Console struct {
	writer  io.Writer
	prefix  string
	postfix string
}

// New constructs a new Console object
func New(writer io.Writer) *Console {
	return &Console{writer, "", ""}
}

// Creates a Console for writing warnings
func (c *Console) Warn() *Console {
	return &Console{writer: c.writer, prefix: yellow, postfix: default_color}
}

// Creates a Console for writing successful output
func (c *Console) Success() *Console {
	return &Console{writer: c.writer, prefix: green, postfix: default_color}
}

// Creates a Console for writing notifications
func (c *Console) Notify() *Console {
	return &Console{writer: c.writer, prefix: blue, postfix: default_color}
}

// Creates a Console for writing errors
func (c *Console) Error() *Console {
	return &Console{writer: c.writer, prefix: red, postfix: default_color}
}

// Printf formats and writes to the console
func (c *Console) Printf(format string, a ...interface{}) (n int, err error) {
	n, err = fmt.Fprintf(c.writer, c.prefix)
	if err != nil {
		return 0, err
	}
	n, err = fmt.Fprintf(c.writer, format, a...)
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintf(c.writer, c.postfix)
	if err != nil {
		return 0, err
	}
	return n, nil
}
