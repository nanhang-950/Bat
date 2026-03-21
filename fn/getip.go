package fn

import (
	"fmt"
	"net"
	"sort"
)

// 获取本地网卡ip
func Getlocalip() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("获取网卡信息失败: %w", err)
	}

	seen := make(map[string]struct{})
	var localIps []string

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			var mask net.IPMask

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
				mask = v.Mask
			case *net.IPAddr:
				ip = v.IP
				mask = ip.DefaultMask()
			}

			if ip == nil || mask == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			if intranetip(ip) {
				cidr := fmt.Sprintf("%s/%d", ip.String(), maskSize(mask))
				if _, ok := seen[cidr]; ok {
					continue
				}
				seen[cidr] = struct{}{}
				localIps = append(localIps, cidr)
			}
		}
	}

	sort.Slice(localIps, func(i, j int) bool {
		return CompareIPs(cidrBaseIP(localIps[i]), cidrBaseIP(localIps[j])) < 0
	})

	return localIps, nil
}

// 判断ip地址为内网ip
func intranetip(ip net.IP) bool {
	return ip != nil && ip.IsPrivate()
}

// 生成网段内所有ip
func GenerateIPs(cidr string) ([]string, error) {
	var ips []string

	err := WalkIPs(cidr, func(ip string) bool {
		ips = append(ips, ip)
		return true
	})
	if err != nil {
		return nil, err
	}

	return ips, nil
}

func EstimateUsableHosts(cidr string) (int, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, err
	}
	if ip.To4() == nil {
		return 0, fmt.Errorf("暂不支持 IPv6 网段: %s", cidr)
	}

	ones, bits := ipNet.Mask.Size()
	if bits != net.IPv4len*8 {
		return 0, fmt.Errorf("无效 IPv4 掩码: %s", cidr)
	}

	hostBits := bits - ones
	switch {
	case hostBits < 0:
		return 0, fmt.Errorf("无效网段: %s", cidr)
	case hostBits == 0:
		return 1, nil
	case hostBits == 1:
		return 2, nil
	default:
		return int((uint64(1) << hostBits) - 2), nil
	}
}

func WalkIPs(cidr string, yield func(string) bool) error {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	if ip.To4() == nil {
		return fmt.Errorf("暂不支持 IPv6 网段: %s", cidr)
	}

	networkIP := cloneIP(ip.Mask(ipNet.Mask))
	ones, bits := ipNet.Mask.Size()

	switch {
	case bits != net.IPv4len*8:
		return fmt.Errorf("无效 IPv4 掩码: %s", cidr)
	case ones == bits:
		yield(networkIP.String())
		return nil
	case ones == bits-1:
		current := cloneIP(networkIP)
		for i := 0; i < 2; i++ {
			if !yield(current.String()) {
				return nil
			}
			IPInc(current)
		}
		return nil
	}

	broadcastIP := lastIP(ipNet)
	for current := cloneIP(networkIP); ipNet.Contains(current); IPInc(current) {
		if current.Equal(networkIP) || current.Equal(broadcastIP) {
			continue
		}
		if !yield(current.String()) {
			return nil
		}
	}

	return nil
}

// ip地址自增
func IPInc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func lastIP(ipNet *net.IPNet) net.IP {
	ip := cloneIP(ipNet.IP)
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j] |= ^ipNet.Mask[j]
	}
	return ip
}

func cloneIP(ip net.IP) net.IP {
	clone := make(net.IP, len(ip))
	copy(clone, ip)
	return clone
}

func maskSize(mask net.IPMask) int {
	size, _ := mask.Size()
	return size
}

func cidrBaseIP(cidr string) string {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return cidr
	}
	return ip.String()
}
