package main

import (
	"bat/fn"
	"bufio"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

type scanTask struct {
	IP   string
	Port int
}

type osResult struct {
	IP string
	OS string
}

func main() {
	fn.Banner()
	fmt.Printf("\n扫描开始，请耐心等待\n")

	start := time.Now()

	cidrs, err := fn.Getlocalip()
	if err != nil {
		fmt.Printf("获取本地网段失败：%v\n", err)
		waitForExit()
		return
	}
	if len(cidrs) == 0 {
		fmt.Println("未发现可用的内网网段，已跳过扫描。")
		waitForExit()
		return
	}

	targetCount := countTargetIPs(cidrs)
	if targetCount == 0 {
		fmt.Println("未生成可扫描的主机地址，已跳过扫描。")
		waitForExit()
		return
	}

	fmt.Printf("发现 %d 个内网网段，待探测地址 %d 个\n", len(cidrs), targetCount)
	aliveIPs := discoverAliveHostsFromCIDRs(cidrs, fn.CommonPorts, 256)
	fmt.Printf("发现 %d 台存活主机\n", len(aliveIPs))

	results := scanOpenPorts(aliveIPs, fn.DefaultPorts, 512)
	results = enrichResultsWithOS(results, 128)
	fmt.Printf("发现 %d 个开放端口\n", len(results))

	aiText, err := fn.ProcessWebSocketData(results)
	if err != nil {
		fmt.Printf("AI 分析已跳过：%v\n", err)
	}

	if err := fn.Savefile(results, aiText, aliveIPs); err != nil {
		fmt.Printf("扫描报告生成失败：%v\n", err)
		waitForExit()
		return
	}

	fmt.Printf("\n扫描报告已生成：内网测绘报告.html\n")
	fmt.Printf("用时： %.2f 秒\n", time.Since(start).Seconds())
	waitForExit()
}

func countTargetIPs(cidrs []string) int {
	total := 0
	for _, cidr := range cidrs {
		count, err := fn.EstimateUsableHosts(cidr)
		if err != nil {
			fmt.Printf("统计网段 %s 失败：%v\n", cidr, err)
			continue
		}
		total += count
	}
	return total
}

func discoverAliveHostsFromCIDRs(cidrs []string, commonPorts []int, maxWorkers int) []string {
	if len(cidrs) == 0 {
		return nil
	}
	if maxWorkers <= 0 {
		maxWorkers = 1
	}

	jobs := make(chan string, maxWorkers*2)
	results := make(chan string, maxWorkers)

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				if fn.IcmpScan(ip) || fn.TcpScan(ip, commonPorts) {
					results <- ip
				}
			}
		}()
	}

	go func() {
		defer close(jobs)

		seen := make(map[string]struct{})
		for _, cidr := range cidrs {
			err := fn.WalkIPs(cidr, func(ip string) bool {
				if _, ok := seen[ip]; ok {
					return true
				}
				seen[ip] = struct{}{}
				jobs <- ip
				return true
			})
			if err != nil {
				fmt.Printf("遍历网段 %s 失败：%v\n", cidr, err)
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	aliveSet := make(map[string]struct{})
	var aliveIPs []string
	for ip := range results {
		if _, ok := aliveSet[ip]; ok {
			continue
		}
		aliveSet[ip] = struct{}{}
		aliveIPs = append(aliveIPs, ip)
	}

	sortIPs(aliveIPs)
	return aliveIPs
}

func scanOpenPorts(ips []string, ports []int, maxWorkers int) []fn.ScanResult {
	if len(ips) == 0 || len(ports) == 0 {
		return nil
	}

	workers := workerCount(len(ips)*len(ports), maxWorkers)
	jobs := make(chan scanTask, workers*2)
	results := make(chan fn.ScanResult, workers*2)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range jobs {
				result, ok := fn.ScanPort(task.IP, task.Port, 2*time.Second)
				if ok {
					results <- result
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, ip := range ips {
			for _, port := range ports {
				jobs <- scanTask{IP: ip, Port: port}
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var openPorts []fn.ScanResult
	for result := range results {
		openPorts = append(openPorts, result)
	}

	sort.Slice(openPorts, func(i, j int) bool {
		if openPorts[i].IP != openPorts[j].IP {
			return ipLess(openPorts[i].IP, openPorts[j].IP)
		}
		return openPorts[i].Port < openPorts[j].Port
	})

	return openPorts
}

func enrichResultsWithOS(results []fn.ScanResult, maxWorkers int) []fn.ScanResult {
	if len(results) == 0 {
		return results
	}
	if maxWorkers <= 0 {
		maxWorkers = 1
	}

	seen := make(map[string]struct{})
	ips := make([]string, 0, len(results))
	for _, result := range results {
		if _, ok := seen[result.IP]; ok {
			continue
		}
		seen[result.IP] = struct{}{}
		ips = append(ips, result.IP)
	}

	workers := workerCount(len(ips), maxWorkers)
	jobs := make(chan string, workers*2)
	osResults := make(chan osResult, workers*2)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				osResults <- osResult{IP: ip, OS: fn.GetOS(ip)}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, ip := range ips {
			jobs <- ip
		}
	}()

	go func() {
		wg.Wait()
		close(osResults)
	}()

	osByIP := make(map[string]string, len(ips))
	for result := range osResults {
		osByIP[result.IP] = result.OS
	}

	for i := range results {
		osName := osByIP[results[i].IP]
		if osName == "" {
			osName = "Unknown"
		}
		results[i].OS = osName
	}

	return results
}

func workerCount(tasks, maxWorkers int) int {
	if tasks <= 0 {
		return 1
	}
	if maxWorkers <= 0 {
		return 1
	}
	if tasks < maxWorkers {
		return tasks
	}
	return maxWorkers
}

func sortIPs(ips []string) {
	sort.Slice(ips, func(i, j int) bool {
		return ipLess(ips[i], ips[j])
	})
}

func ipLess(left, right string) bool {
	return fn.CompareIPs(left, right) < 0
}

func waitForExit() {
	fmt.Println("按回车键退出...")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
}
