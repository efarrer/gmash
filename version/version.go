package version

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
)

const (
	url    = "https://raw.githubusercontent.com/efarrer/gmash/blob/master/version/version.go"
	String = "0.0.1"
)

func GetLatestVersion() (string, error) {
	return getLatestVersion(http.Get)
}

func getLatestVersion(get func(string) (*http.Response, error)) (string, error) {
	resp, err := get(url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Invalid response %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile("String = \"(.*)\"")
	matches := re.FindSubmatch(body)
	if len(matches) != 2 {
		return "", fmt.Errorf("Unable to find version in \"%s\"", string(body))
	}

	version := string(matches[1])

	return version, nil
}
