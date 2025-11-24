package fn

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// 扫描给定的ip地址的端口

func Scan(ip string, portsTask chan int, results chan ScanResult, wg *sync.WaitGroup) {
	defer wg.Done()

	//从通道中获取端口
	for {
		portTask, ok := <-portsTask

		if !ok {
			break
		}

		//构建地址字符串
		address := fmt.Sprintf("%s:%d", ip, portTask)

		//使用DialTimeout尝试连接目标地址，超时时间设置为6秒
		conn, err := net.DialTimeout("tcp", address, time.Second*6)

		result := ScanResult{
			IP:       ip,
			Port:     portTask,
			Protocol: GetProtocol(portTask),
			OS:       GetOS(ip),
			Bb:       "1",
		}

		//如果连接成功，打印地址和状态并关闭连接
		if err == nil {
			//state := "开放"
			//显示ip和开放端口
			//fmt.Println(address, state)
			conn.Close()
		} else {
			continue
		}

		results <- result
	}
}
