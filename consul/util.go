package consul

import (
	"encoding/binary"
	"fmt"
	"github.com/hashicorp/serf/serf"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

/*
 * Contains an entry for each private block:
 * 10.0.0.0/8
 * 172.16.0.0/12
 * 192.168/16
 */
var privateBlocks []*net.IPNet

// serverparts is used to return the parts of a server role
type serverParts struct {
	Datacenter string
	Port       int
	Bootstrap  bool
}

func init() {
	// Add each private block
	privateBlocks = make([]*net.IPNet, 3)
	_, block, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		panic(fmt.Sprintf("Bad cidr. Got %v", err))
	}
	privateBlocks[0] = block

	_, block, err = net.ParseCIDR("172.16.0.0/12")
	if err != nil {
		panic(fmt.Sprintf("Bad cidr. Got %v", err))
	}
	privateBlocks[1] = block

	_, block, err = net.ParseCIDR("192.168.0.0/16")
	if err != nil {
		panic(fmt.Sprintf("Bad cidr. Got %v", err))
	}
	privateBlocks[2] = block
}

// strContains checks if a list contains a string
func strContains(l []string, s string) bool {
	for _, v := range l {
		if v == s {
			return true
		}
	}
	return false
}

// ensurePath is used to make sure a path exists
func ensurePath(path string, dir bool) error {
	if !dir {
		path = filepath.Dir(path)
	}
	return os.MkdirAll(path, 0755)
}

// Returns if a member is a consul server. Returns a bool,
// the data center, and the rpc port
func isConsulServer(m serf.Member) (bool, *serverParts) {
	if m.Tags["role"] != "consul" {
		return false, nil
	}

	datacenter := m.Tags["dc"]
	port_str := m.Tags["port"]
	_, bootstrap := m.Tags["bootstrap"]
	port, err := strconv.Atoi(port_str)
	if err != nil {
		return false, nil
	}

	return true, &serverParts{datacenter, port, bootstrap}
}

// Returns if a member is a consul node. Returns a boo,
// and the data center.
func isConsulNode(m serf.Member) (bool, string) {
	if m.Tags["role"] != "node" {
		return false, ""
	}
	return true, m.Tags["dc"]
}

// Returns if the given IP is in a private block
func isPrivateIP(ip_str string) bool {
	ip := net.ParseIP(ip_str)
	for _, priv := range privateBlocks {
		if priv.Contains(ip) {
			return true
		}
	}
	return false
}

// GetPrivateIP is used to return the first private IP address
// associated with an interface on the machine
func GetPrivateIP() (*net.IPNet, error) {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("Failed to get interface addresses: %v", err)
	}

	// Find private IPv4 address
	for _, addr := range addresses {
		ip, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ip.IP.To4() == nil {
			continue
		}
		if !isPrivateIP(ip.IP.String()) {
			continue
		}
		return ip, nil
	}
	return nil, fmt.Errorf("No private IP address found")
}

// Converts bytes to an integer
func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// Converts a uint to a byte slice
func uint64ToBytes(u uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, u)
	return buf
}

// runtimeStats is used to return various runtime information
func runtimeStats() map[string]string {
	return map[string]string{
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"version":    runtime.Version(),
		"max_procs":  strconv.FormatInt(int64(runtime.GOMAXPROCS(0)), 10),
		"goroutines": strconv.FormatInt(int64(runtime.NumGoroutine()), 10),
		"cpu_count":  strconv.FormatInt(int64(runtime.NumCPU()), 10),
	}
}
