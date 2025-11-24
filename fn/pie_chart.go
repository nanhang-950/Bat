package fn

import (
	"fmt"
	"os"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func PieChart(f *os.File, results map[string][]ScanResult) {
	f.WriteString(`
		<h1 style="text-align: center;">内网测绘报告</h1>
		<title>内网测绘报告</title>
    `)

	// 创建并配置第一个饼状图
	pie1 := charts.NewPie()
	pie1.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
		Title: "端口分布",
	}))
	ports := make(map[int]int)
	protocols := make(map[string]int)

	for _, scanResults := range results {
		for _, result := range scanResults {
			ports[result.Port]++
			protocols[result.Protocol]++
		}
	}

	var portData []opts.PieData
	for port, count := range ports {
		portData = append(portData, opts.PieData{
			Name:  fmt.Sprintf("%d", port),
			Value: count,
		})
	}

	pie1.AddSeries("端口", portData).SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Position:  "outside",
			Formatter: "{b}: {d}%", // 显示百分比
		}),
	)

	// 创建并配置第二个饼状图
	pie2 := charts.NewPie()
	pie2.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
		Title: "协议分布",
	}))
	var protocolData []opts.PieData
	for protocol, count := range protocols {
		protocolData = append(protocolData, opts.PieData{
			Name:  protocol,
			Value: count,
		})
	}

	pie2.AddSeries("协议", protocolData).SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Position:  "outside",
			Formatter: "{b}: {d}%", // 显示百分比
		}),
	)

	// 渲染两个饼状图
	pie1.Render(f)
	f.WriteString("<hr>") // 添加分隔符
	pie2.Render(f)
	f.WriteString("<hr>") // 添加分隔符
}
