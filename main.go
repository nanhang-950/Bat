package main

import (
	"bat/fn"
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

func main() {
	fn.Banner()
	fmt.Printf("\n扫描开始，请耐心等待\n")

	// 获取网卡ip
	cidrs := fn.Getlocalip()

	// 定义一个用于存储ip的切片
	var allIPs []string

	// 创建一个新的进度条 并在接下来分成三份来进行进度条累加
	bar := progressbar.Default(int64(len(cidrs) * 100))
	barLength := (len(cidrs) * 100) / 4 // 用于循环的次数
	count := 0

	// 遍历每个cidr地址段
	for _, cidr := range cidrs {
		ips, err := fn.GenerateIPs(cidr)
		if err != nil {
			fmt.Println("错误:", err)
			continue
		}
		allIPs = append(allIPs, ips...)
	}

	//第一次循环累加进度条
	for i := 0; i < barLength; i++ {
		count++
		time.Sleep(10 * time.Millisecond) //假设任务耗时
		bar.Add(1)
		if count == barLength {
			count = 0
			break
		}
	}

	// 常见端口列表
	var commonPorts = []int{80, 443, 22, 21, 3389, 25, 23, 137, 138, 139, 3389}

	// 并发扫描
	var wg sync.WaitGroup
	results1 := make(chan string, len(allIPs))

	// 并发数
	const maxConcurrency = 700
	semaphore := make(chan struct{}, maxConcurrency)

	start := time.Now()

	// 扫描 IP 地址
	for _, ip := range allIPs {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			semaphore <- struct{}{}        // 请求一个槽位
			defer func() { <-semaphore }() // 释放一个槽位

			//默认使用Icmp扫描存活，如果Icmp失败则使用Tcp
			if fn.IcmpScan(ip) {
				results1 <- ip
			} else if fn.TcpScan(ip, commonPorts) {
				results1 <- ip
			}
		}(ip)
	}

	go func() {
		wg.Wait()
		close(results1)
	}()

	//第二次循环累加进度条
	for i := 0; i < barLength; i++ {
		count++
		time.Sleep(100 * time.Millisecond) //假设任务耗时
		bar.Add(1)
		if count == barLength {
			count = 0
			break
		}
	}

	var aliveIPs []string

	for ip := range results1 {
		aliveIPs = append(aliveIPs, ip)
	}

	results2 := make(chan fn.ScanResult, len(aliveIPs)*len(fn.DefaultPorts))
	results2Copy := make(chan fn.ScanResult, len(aliveIPs)*len(fn.DefaultPorts))

	for _, ip := range aliveIPs {
		portsTask := make(chan int, len(fn.DefaultPorts))
		for _, port := range fn.DefaultPorts {
			portsTask <- port
		}
		close(portsTask)

		for threads := 0; threads < 600; threads++ {
			wg.Add(1)
			go fn.Scan(ip, portsTask, results2, &wg)
			wg.Add(1)
			go fn.Scan(ip, portsTask, results2Copy, &wg)
		}
	}

	//第三次循环累加进度条
	for i := 0; i < barLength; i++ {
		count++
		time.Sleep(10 * time.Millisecond) //假设任务耗时
		bar.Add(1)
		if count == barLength {
			count = 0
			break
		}
	}

	go func() {
		wg.Wait()
		close(results2)
		close(results2Copy)
	}()

	// 使用全局通道 调用ai扫描接口 来传递数据给AiInterface
	fn.ProcessWebSocketData(results2Copy)

	// 生成报告文件
	fn.Savefile(results2)

	// 更新进度条至 100%
	bar.Finish()

	// 输出扫描结果信息
	fmt.Printf("\n扫描报告已生成：内网测绘报告.html\n")
	fmt.Printf("用时： %.2f 秒\n", time.Since(start).Seconds())
	fmt.Println("按回车键退出...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}
