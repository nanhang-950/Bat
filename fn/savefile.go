package fn

import (
	"fmt"
	"html/template"
	"os"
)

// 定义一全局个映射类型的变量
var processedResults map[string][]ScanResult

// Savefile 将扫描结果格式化为HTML并写入到result.html文件中
func Savefile(results chan ScanResult) {
	IpResults := make(map[string][]ScanResult)

	for result := range results {
		IpResults[result.IP] = append(IpResults[result.IP], result)
	}

	// 将处理后的数据保存到全局变量
	processedResults = IpResults

	// 创建html文件
	file, err := os.Create("内网测绘报告.html")
	if err != nil {
		fmt.Println("创建文件失败：", err)
		return
	}

	PieChart(file, IpResults)
	defer file.Close()

	// 使用 GetAiText 获取AiText切片
	text := GetAiText()

	// 定义一个html模板，用于格式化扫描结果
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>测绘报告</title>
	<style>
		table {
			width: 100%;
			max-width: 2000px; /* 设置最大宽度 */
			border-collapse: collapse;
			margin: auto; /* 居中对齐 */
			table-layout: fixed; /* 固定列宽 */
		}
		th, td {
			border: 1px solid black;
			padding: 8px;
			text-align: left;
			overflow: hidden; /* 隐藏超出部分 */
			text-overflow: ellipsis; /* 超出部分显示省略号 */
		}
		th {
			background-color: #4CAF50;
			color: white;
		}
		th:first-child, td:first-child {
			width: 20%; /* 固定宽度 */
		}
		th:nth-child(2), td:nth-child(2) {
			width: 80%; /* 固定宽度 */
		}
		.btm{
			border: 2px #000 solid;
			border-radius: 1em;
			margin: 30px 40px 0 40px;
		}
		.btm p{
			text-indent: 2em;
			margin: 15px 30px 10px 30px;
			font-size: 1.2em;
			line-height: 35px;
		}

	</style>
</head>
<body>
	{{range $ip, $results := .Results}}
		<h2>IP: {{$ip}}</h2>
		{{/* 如果所有结果都具有相同的OS，则显示第一个结果的OS */}}
		{{with index $results 0}}
		<h2>操作系统: {{.OS}}</h2>
		{{end}}
		<table>
			<tr>
				<th>端口</th>
				<th>服务</th>
			</tr>
			{{range $results}}
			<tr>
				<td>{{.Port}}</td>
				<td>{{.Protocol}}</td>
			</tr>
			{{end}}
		</table>
	{{end}}
	<h1 style="text-align: center;margin-top: 50px;">大模型分析报告</h1>
		<div class="btm">
		{{range .AiText}}
		<p>{{.}}</p>
		{{end}}
		</div>
		<div style= "height:100px;"></div>
	</div>
</body>
</html>`

	// 使用template.New创建一个新的模板对象，并解析定义的HTML模板tmpl
	t, err := template.New("webpage").Parse(tmpl)
	if err != nil {
		fmt.Println("无法解析模板：", err)
		panic(err)
	}

	// 创建一个map来存储数据
	data := struct {
		Results map[string][]ScanResult
		AiText  []string
	}{
		Results: IpResults,
		AiText:  text,
	}

	// 使用Execute将ipResults数据填充到模板中，并写入到文件中
	if err := t.Execute(file, data); err != nil {
		fmt.Println("模板执行失败：", err)
	}

}

// 定义一个方法返回处理后的结果给调用者
func GetProcessedResults() map[string][]ScanResult {
	return processedResults
}
