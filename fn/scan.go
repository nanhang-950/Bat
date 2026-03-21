package fn

import (
	"net"
	"strconv"
	"time"
)

func ScanPort(ip string, port int, timeout time.Duration) (ScanResult, bool) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return ScanResult{}, false
	}
	defer conn.Close()

	return ScanResult{
		IP:       ip,
		Port:     port,
		Protocol: GetProtocol(port),
		OS:       "Unknown",
		Bb:       "1",
	}, true
}
