package payload

import (
	"encoding/binary"
	"fmt"
)

type parser func([]byte) (interface{}, []byte, error)

func parseUInt32(b []byte) (interface{}, []byte, error) {
	if len(b) < 4 {
		return nil, []byte{}, fmt.Errorf("Unable to parse uint32 not enough bytes remaining (%d)", len(b))
	}

	return binary.BigEndian.Uint32(b), b[4:], nil
}

func parseBytes(b []byte) (interface{}, []byte, error) {
	return b, []byte{}, nil
}

func parseString(b []byte) (interface{}, []byte, error) {
	length, rest, err := parseUInt32(b)
	if err != nil {
		return "", []byte{}, err
	}
	lenInt := length.(uint32)
	b = rest
	if uint32(len(b)) < lenInt {
		return "", []byte{}, fmt.Errorf("Unable to parse string not enough bytes remaining (%d)", len(b))
	}

	return string(b[:lenInt]), b[lenInt:], nil
}

func parsePayload(b []byte, parsers []parser) ([]interface{}, error) {
	ress := make([]interface{}, 0, len(parsers))

	for p := range parsers {
		res, rest, err := parsers[p](b)
		if err != nil {
			return nil, err
		}
		ress = append(ress, res)
		b = rest
	}
	return ress, nil
}

// PtyConfig contains PTY configuration from a Pty request
type PtyConfig struct {
	ttyType       string
	widthChars    uint32
	heightCols    uint32
	widthPixels   uint32
	heightPixels  uint32
	terminalModes []byte
}

// ParsePtyReq Parses the SSH Pty request payload
func ParsePtyReq(b []byte) (*PtyConfig, error) {
	// See RFC 4254 6.2
	data, err := parsePayload(b, []parser{parseString, parseUInt32, parseUInt32, parseUInt32, parseUInt32, parseBytes})
	if err != nil {
		return nil, err
	}

	return &PtyConfig{
		ttyType:       data[0].(string),
		widthChars:    data[1].(uint32),
		heightCols:    data[2].(uint32),
		widthPixels:   data[3].(uint32),
		heightPixels:  data[4].(uint32),
		terminalModes: data[5].([]byte),
	}, nil
}
