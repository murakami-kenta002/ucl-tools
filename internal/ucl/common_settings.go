package ucl

import (
	"errors"
	"net"
	"os"
	"strings"
)

type MultiFlag []string

func (m *MultiFlag) String() string {
	return strings.Join(*m, ", ")
}
func (m *MultiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func isIpv4(ip string) bool {
	if net.ParseIP(ip) != nil {
		for i := 0; i < len(ip); i++ {
			switch ip[i] {
			case '.':
				return true
			case ':':
				return false
			}
		}
	}

	return false
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetIpAddrsOfAllInterfaces() ([]string, error) {

	var ipaddrs []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return ipaddrs, err
	}

	/* make ipaddrs slice */
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			ipv4 := ip.To4()
			if ipv4 != nil {
				ipaddrs = append(ipaddrs, ipv4.String())
			}
		}
	}

	if len(ipaddrs) == 0 {
		return ipaddrs, errors.New("Cannot Find My IpAddr from ifaces")
	} else {
		return ipaddrs, nil
	}

}

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
