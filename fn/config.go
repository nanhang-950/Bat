package fn

import (
	"bytes"
	"fmt"
	"net"
	"sync"
)

// 定义一个结构体类型，用于存储每个扫描结果，包括IP、端口号、协议和状态
type ScanResult struct {
	IP       string
	Port     int
	Protocol string
	OS       string
	Bb       string
}

var CommonPorts = []int{22, 111, 135, 139, 445, 514, 631, 3389, 5985}

var OsCache sync.Map

// 常见的端口号的切片
var DefaultPorts = []int{7, 19, 20, 21, 22, 23, 25, 42, 43, 49, 53, 67, 68, 69, 70, 79, 80, 81, 88, 102, 110, 113, 119, 123, 135, 139, 143, 161, 162, 194, 443, 445, 514, 1433, 1521, 1701, 1723, 1883, 3306, 3389, 5100, 5421, 5432, 5900, 5984, 6379, 6667, 7001, 7002, 7680, 8000, 8080, 8001, 8082, 8089, 9000, 9092, 9100, 9200, 10000, 11211, 27017, 50000}

func Banner() {
	bat := `
  ██████╗  █████╗ ████████╗
  ██╔══██╗██╔══██╗╚══██╔══╝
  ██████╔╝███████║   ██║
  ██╔══██╗██╔══██║   ██║
  ██████╔╝██║  ██║   ██║
  ╚═════╝ ╚═╝  ╚═╝   ╚═╝
    BAT version: 1.0.0
`
	fmt.Print(bat)
}

func CompareIPs(left, right string) int {
	leftIP := net.ParseIP(left).To4()
	rightIP := net.ParseIP(right).To4()

	switch {
	case leftIP == nil && rightIP == nil:
		if left < right {
			return -1
		}
		if left > right {
			return 1
		}
		return 0
	case leftIP == nil:
		return 1
	case rightIP == nil:
		return -1
	default:
		return bytes.Compare(leftIP, rightIP)
	}
}
