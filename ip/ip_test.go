package ip

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeRouter(r string, err error) netRouter {
	return func() ([]byte, error) {
		return []byte(r), err
	}
}

func TestDefaultIface_HandlesValidInput(t *testing.T) {
	route := `
Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
default         ip-10-211-55-1. 0.0.0.0         UG    100    0        0 enp0s5
10.37.129.0     *               255.255.255.0   U     100    0        0 enp0s6
10.211.55.0     *               255.255.255.0   U     100    0        0 enp0s5
link-local      *               255.255.0.0     U     1000   0        0 enp0s6
172.17.0.0      *               255.255.0.0     U     0      0        0 docker0
`
	iface, err := defaultIface(makeRouter(route, nil))
	assert.NoError(t, err)
	assert.Equal(t, "enp0s5", iface)
}

func TestDefaultIface_HandlesRouterError(t *testing.T) {
	_, err := defaultIface(makeRouter("", errors.New("")))
	assert.Error(t, err)
}

func TestDefaultIface_CantFindInterface(t *testing.T) {
	_, err := defaultIface(makeRouter("", nil))
	assert.Error(t, err)
}

func makeInterfaces(ifaces []net.Interface, err error) netInterfaces {
	return func() ([]net.Interface, error) {
		return ifaces, err
	}
}

func TestGetIP_ReturnsErrorIfIfaceNotFound(t *testing.T) {
	_, err := getIP("nope", makeInterfaces([]net.Interface{}, nil))
	assert.Error(t, err)
}

func TestGetIP_ReturnsErrorIfCantGetInterfaces(t *testing.T) {
	_, err := getIP("", makeInterfaces([]net.Interface{}, errors.New("")))
	assert.Error(t, err)
}

func TestGetIP_ReturnsIpAddress(t *testing.T) {
	ifaces, err := net.Interfaces()
	assert.NoError(t, err)

	addr, err := getIP(ifaces[0].Name, makeInterfaces(ifaces, nil))
	assert.NoError(t, err)
	assert.NotEqual(t, addr, "")
}

func TestGetPublicIP_FailedRouter(t *testing.T) {
	_, err := getPublicIP(makeRouter("", errors.New("")), makeInterfaces([]net.Interface{}, nil))
	assert.Error(t, err)
}

func TestLinusPublicIP(t *testing.T) {
	ip, err := LinuxPublicIP()
	assert.NoError(t, err)
	assert.NotEqual(t, "", ip)
}
