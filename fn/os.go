package fn

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var ttlPattern = regexp.MustCompile(`(?i)ttl[=\s:]+(\d+)`)

// GetOs 根据 TTL 值推断操作系统
func GetOs(ttl int) string {
	switch {
	case ttl >= 128:
		return "Windows"
	case ttl >= 64:
		return "Linux"
	case ttl >= 60:
		return "AIX"
	case ttl >= 32:
		return "Cisco Router"
	default:
		return "Unknown"
	}
}

// PingAndDetectOS 发送 ICMP 请求并检测操作系统
func GetOS(ip string) string {
	if cached, ok := OsCache.Load(ip); ok {
		if osName, ok := cached.(string); ok {
			return osName
		}
	}

	osName := detectOS(ip)
	OsCache.Store(ip, osName)
	return osName
}

func detectOS(ip string) string {
	ttl := pingTTL(ip)
	if ttl > 0 {
		return GetOs(ttl)
	}

	switch {
	case TcpScan(ip, []int{22, 111, 514, 631}):
		return "Linux"
	case TcpScan(ip, []int{135, 139, 445, 3389, 5985}):
		return "Windows"
	default:
		return "Unknown"
	}
}

func pingTTL(ip string) int {
	output, err := exec.Command("ping", pingArgs(ip)...).CombinedOutput()
	if err != nil && len(output) == 0 {
		return 0
	}

	matches := ttlPattern.FindStringSubmatch(strings.ToLower(string(output)))
	if len(matches) != 2 {
		return 0
	}

	ttl := 0
	for _, ch := range matches[1] {
		ttl = ttl*10 + int(ch-'0')
	}
	return ttl
}

func pingArgs(ip string) []string {
	if runtime.GOOS == "windows" {
		return []string{"-n", "1", "-w", "1000", ip}
	}
	return []string{"-c", "1", "-W", "1", ip}
}
