package fn

import (
	"fmt"
	"net"
	"time"

	"github.com/tatsushid/go-fastping"
)

func IcmpScan(ip string) bool {
	p := fastping.NewPinger()
	ra, err := net.ResolveIPAddr("ip4:icmp", ip)
	if err != nil {
		return false
	}
	p.AddIPAddr(ra)

	found := false
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		found = true
	}
	p.OnIdle = func() {}
	p.MaxRTT = time.Second

	if err := p.Run(); err != nil {
		return false
	}

	return found
}

// TCP扫描
func TcpScan(ip string, ports []int) bool {
	for _, port := range ports {
		addr := fmt.Sprintf("%s:%d", ip, port)
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}
