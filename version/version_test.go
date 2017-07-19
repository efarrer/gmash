package version

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
)

func Test_GetLatestVersion_HandlesGETFailure(t *testing.T) {
	_, err := getLatestVersion(func(string) (*http.Response, error) { return nil, fmt.Errorf("Some error") })
	assert.Error(t, err)
}

func getResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: ioutil.NopCloser(strings.NewReader(body))}
}

func Test_GetLatestVersion_HandlesStatusFailure(t *testing.T) {
	_, err := getLatestVersion(func(string) (*http.Response, error) { return getResponse(http.StatusNotFound, ""), nil })
	assert.Error(t, err)
}

func Test_GetLatestVersion_HandlesReadFailure(t *testing.T) {
	resp := getResponse(http.StatusOK, "")
	// The second read will fail
	resp.Body = ioutil.NopCloser(iotest.TimeoutReader(resp.Body))
	ioutil.ReadAll(resp.Body)
	_, err := getLatestVersion(func(string) (*http.Response, error) { return resp, nil })
	assert.Error(t, err)
}

func Test_GetLatestVersion_HandlesParseFailure(t *testing.T) {
	_, err := getLatestVersion(func(string) (*http.Response, error) {
		return getResponse(http.StatusOK, "	Barf = \"0.0.1\""), nil
	})
	assert.Error(t, err)
}

func Test_GetLatestVersion_Works(t *testing.T) {
	version, err := getLatestVersion(func(string) (*http.Response, error) {
		return getResponse(http.StatusOK, "	String = \"0.0.1\""), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, version, "0.0.1")
}
