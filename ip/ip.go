package ip

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"os/exec"
	"strings"
)

type netRouter func() ([]byte, error)

func defaultIface(get netRouter) (string, error) {
	output, err := get()
	if err != nil {
		return "", err
	}

	lineScanner := bufio.NewScanner(bytes.NewReader(output))

	for lineScanner.Scan() {
		line := lineScanner.Text()

		rightLine := false
		wordScanner := bufio.NewScanner(strings.NewReader(line))
		wordScanner.Split(bufio.ScanWords)
		for w := 0; wordScanner.Scan(); w++ {
			word := wordScanner.Text()
			if w == 0 && word == "default" {
				rightLine = true
			}
			if rightLine && w == 7 {
				return string(word), nil
			}
		}
		if err := wordScanner.Err(); err != nil {
			return "", err
		}
	}
	if err := lineScanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("unable to find default interface")
}

type netInterfaces func() ([]net.Interface, error)

func getIP(ifaceName string, get netInterfaces) (string, error) {
	ifaces, err := get()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Name == ifaceName {
			addrs, err := iface.Addrs()
			if err != nil {
				return "", err
			}
			for _, addr := range addrs {
				switch v := addr.(type) {
				case *net.IPNet:
					return v.IP.String(), nil
				case *net.IPAddr:
					return v.IP.String(), nil
				}
			}
		}
	}
	return "", errors.New("unable to find ip for the interface")
}

func getPublicIP(getRouter netRouter, getIface netInterfaces) (string, error) {
	iface, err := defaultIface(getRouter)
	if err != nil {
		return "", err
	}
	return getIP(iface, getIface)
}

func LinuxPublicIP() (string, error) {
	return getPublicIP(exec.Command("/sbin/route").Output, net.Interfaces)
}
