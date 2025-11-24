package fn

import (
	"fmt"
	"net"
	"os"
)

// 获取本地网卡ip
func Getlocalip() []string {

	//获取本地所有网络接口
	interfaces, err := net.Interfaces()

	//错误处理
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	//定义一个切片用于存储ip
	var localIps []string

	//迭代网络接口
	for _, iface := range interfaces {

		//使用Addrs获取每个网络接口的ip
		addrs, err := iface.Addrs()

		//错误处理
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		//迭代ip地址
		for _, addr := range addrs {

			//定义ip变量
			var ip net.IP
			var mask net.IPMask
			//判断ip地址的类型
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
				mask = v.Mask
			case *net.IPAddr:
				ip = v.IP
				mask = ip.DefaultMask()
			}

			//如果ip为空或ip为回环地址或ip不为ipv4地址则
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			//判断是否为内网ip并添加到切片
			if intranetip(ip) {
				cidr := fmt.Sprintf("%s/%d", ip.String(), maskSize(mask))
				localIps = append(localIps, cidr)
			}
		}
	}

	//ip切片
	return localIps
}

// 判断ip地址为内网ip
func intranetip(ip net.IP) bool {
	//定义内网地址列表
	blocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	//遍历内网ip地址块
	for _, cidr := range blocks {

		//解析CIDR并判断是否包含ip
		_, block, _ := net.ParseCIDR(cidr)

		//检查给定的ip地址是否存在block定义的网络范围内
		//如果在返回true，即为内网ip
		if block.Contains(ip) {
			return true
		}
	}

	//如果不在返回false
	return false
}

// 生成网段内所有ip
func GenerateIPs(cidr string) ([]string, error) {
	var ips []string

	ip, ipNet, err := net.ParseCIDR(cidr)

	//错误处理
	if err != nil {
		return nil, err
	}

	//生成网段内所有ip地址
	//将起始ip与网络掩码进行按位与操作，得到网络的第一个ip地址
	//检查ip地址是否在ip网络ipNet范围内
	//然后自增ip地址
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); IPInc(ip) {
		//每次循环将生成{
		if !ip.Equal(ipNet.IP) && !ip.Equal(lastIP(ipNet)) {
			ips = append(ips, ip.String())
		}
	}
	//如果生成的ip地址数量大于2，将第一个和最后一个ip地址去掉并返回，因为这些通常是网络地址和广播地址
	if len(ips) > 2 {
		return ips[1 : len(ips)-1], nil
	}

	return ips, nil
}

// ip地址自增
func IPInc(ip net.IP) {
	//使用for循环从ip地址的最后一部分开始自增
	for j := len(ip) - 1; j >= 0; j-- {
		//对ip地址的最后一部分进行自增
		ip[j]++
		//如果自增后不产生进位，则跳出循环
		if ip[j] > 0 {
			break
		}
	}
}

func lastIP(ipNet *net.IPNet) net.IP {
	ip := ipNet.IP
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j] |= ^ipNet.Mask[j]
	}
	return ip
}

func maskSize(mask net.IPMask) int {
	size, _ := mask.Size()
	return size
}
