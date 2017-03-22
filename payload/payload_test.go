package payload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUint32_ParsesBytes(t *testing.T) {
	res, rest, err := parseUInt32([]byte{0x0, 0x0, 0x0, 0x1})
	assert.Equal(t, uint32(1), res.(uint32))
	assert.Equal(t, rest, []byte{})
	assert.Nil(t, err)
}

func TestParseUint32_ReturnsErrorIfNotEnoughBytes(t *testing.T) {
	res, rest, err := parseUInt32([]byte{0x0})
	assert.Nil(t, res)
	assert.Equal(t, rest, []byte{})
	assert.NotNil(t, err)
}

func TestParseUint32_ReturnsUnusedBytes(t *testing.T) {
	_, rest, err := parseUInt32([]byte{0x0, 0x0, 0x0, 0x0, 0x1})
	assert.Equal(t, rest, []byte{0x1})
	assert.Nil(t, err)
}

func TestParseBytes_ReturnsUnusedBytes(t *testing.T) {
	res, rest, err := parseBytes([]byte{0x0, 0x0, 0x0, 0x0, 0x1})
	assert.Equal(t, res, []byte{0x0, 0x0, 0x0, 0x0, 0x1})
	assert.Equal(t, rest, []byte{})
	assert.Nil(t, err)
}

func TestParseString_ReturnsParsedString(t *testing.T) {
	res, rest, err := parseString([]byte{0x0, 0x0, 0x0, 0x1, 0x3B})
	assert.Equal(t, res.(string), ";")
	assert.Equal(t, rest, []byte{})
	assert.Nil(t, err)
}

func TestParseString_ReturnsErrorIfNotEnoughBytes(t *testing.T) {
	res, rest, err := parseString([]byte{0x0})
	assert.Equal(t, res.(string), "")
	assert.Equal(t, rest, []byte{})
	assert.NotNil(t, err)
}

func TestParseString_ReturnsErrorIfStringTooShort(t *testing.T) {
	res, rest, err := parseString([]byte{0x0, 0x0, 0x0, 0x1})
	assert.Equal(t, res.(string), "")
	assert.Equal(t, rest, []byte{})
	assert.NotNil(t, err)
}

func TestParsePtyRequst_HandlesValidPtyRequest(t *testing.T) {
	ptyConf, err := ParsePtyReq([]byte{
		0x0, 0x0, 0x0, 0x1, 0x3b, // tty type
		0x0, 0x0, 0x0, 0x0A, // width chars
		0x0, 0x0, 0x0, 0xA0, // height columns
		0x0, 0x0, 0x0, 0x0, // width pixels
		0x0, 0x0, 0x0, 0x0, // height pixesl
		0x0, 0x0, 0x0, 0x0, // terminal modes
	})

	assert.Nil(t, err)
	assert.Equal(t, *ptyConf, PtyConfig{";", uint32(10), uint32(160), uint32(0), uint32(0), []byte{0x0, 0x0, 0x0, 0x0}})
}

func TestParsePtyRequst_HandlesInValidPtyRequest(t *testing.T) {
	ptyConf, err := ParsePtyReq([]byte{
		0x0, 0x0, 0x0, 0x1, 0x3b, // tty type
		0x0, 0x0, 0x0, 0x0A, // width chars
		0x0, 0x0, 0x0, 0xA0, // height columns
		0x0, 0x0, 0x0, 0x0, // width pixels
		0x0, 0x0, 0x0, // Whoops missing a byte
	})

	assert.NotNil(t, err)
	assert.Nil(t, ptyConf)
}
