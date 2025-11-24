package fn

import (
	"os/exec"
	"strconv"
	"strings"
)

// GetOs 根据 TTL 值推断操作系统
func GetOs(ttl int) string {
	switch {
	case ttl >= 128:
		return "Windows"
	case ttl >= 64 && ttl < 128:
		return "Linux"
	case ttl >= 32 && ttl < 64:
		return "Cisco Router"
	case ttl >= 60 && ttl < 64:
		return "AIX"
	default:
		return "Unknown"
	}
}

// PingAndDetectOS 发送 ICMP 请求并检测操作系统
func GetOS(ip string) string {
	cmd := exec.Command("ping", "-c", "1", "-W", "1", ip)
	output, err := cmd.CombinedOutput()
  if err != nil {
		for _, port := range CommonPorts {
			if (port == 22 || port == 631 || port == 514 || port == 111) ||
				(port == 135 || port == 139 || port == 445 || port == 3389 || port == 5985) {
				if TcpScan(ip, []int{port}) {
					if port == 22 || port == 631 || port == 514 || port == 111 {
						return "Linux"
					}
					if port == 135 || port == 139 || port == 445 || port == 3389 || port == 5985 {
						return "Windows"
					}
				}
			}
		}
  }
	lines := strings.Split(string(output), "\n")
	var ttl int
	for _, line := range lines {
		if strings.Contains(line, "ttl=") {
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.HasPrefix(field, "ttl=") {
					ttlStr := strings.TrimPrefix(field, "ttl=")
					ttl, err = strconv.Atoi(ttlStr)
					if err != nil {
						return ""
					}
					break
				}
			}
			break
		}
	}

	if ttl == 0 {
		return "Unknown"
	}

	os := GetOs(ttl)
	return os
}
