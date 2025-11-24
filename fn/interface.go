package fn

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

/*
   WebAPI 接口调用示例 接口文档（必看）：https://www.xfyun.cn/doc/spark/Web.html
  错误码链接：https://www.xfyun.cn/doc/spark/%E6%8E%A5%E5%8F%A3%E8%AF%B4%E6%98%8E.html（code返回错误码时必看）
*/

var aiText []string

var (
	hostUrl   = "wss://spark-api.xf-yun.com/v3.5/chat"
	appid     = "9bcdfbc5"
	apiSecret = "NDZhNzZjNzRmYTU0ZDQ4NmNmM2NlYmY2"
	apiKey    = "c2b637d838664be80412a417cf145377"
)

func ProcessWebSocketData(results2 chan ScanResult) {
	d := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, resp, err := d.Dial(assembleAuthUrl1(hostUrl, apiKey, apiSecret), nil)

	if err != nil {
		fmt.Printf("无法建立连接：%v\n响应：%s\n", err, readResp(resp))
		return
	}

	defer conn.Close()

	resultsStr := channelToString(results2)
	question := resultsStr
	go send(conn, appid, question)

	var answer string

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("读取消息错误：", err)
			break
		}

		content, err := parseMessage(msg)
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
			return
		}

		answer += content
		if Answer(content) {
			break
		}
	}

	aiText = strings.Split(answer, "\n")

}

func send(conn *websocket.Conn, appid, question string) {
	data := genParams1(appid, question)
	if err := conn.WriteJSON(data); err != nil {
		fmt.Println("发生数据失败", err)
	}
}

func parseMessage(msg []byte) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg, &data); err != nil {
		return "", err
	}
	payload := data["payload"].(map[string]interface{})
	choices := payload["choices"].(map[string]interface{})
	header := data["header"].(map[string]interface{})
	code := header["code"].(float64)

	if code != 0 {
		return "", fmt.Errorf("代码错误 %v: %v", code, data["payload"])
	}

	status := choices["status"].(float64)
	text := choices["text"].([]interface{})
	content := text[0].(map[string]interface{})["content"].(string)

	if status == 2 {
		return content, nil
	}

	return content, nil
}

func Answer(content string) bool {
	return false
}

// 生成参数
func genParams1(appid, question string) map[string]interface{} { // 根据实际情况修改返回的数据结构和字段名

	messages := []Message{
		{Role: "system", Content: "根据以下内网测绘的结果，请帮我评估这个内网的安全性，请先分点给出建议，并在最后总结一段输出文字"}, //设置对话背景或者模型角色
		{Role: "user", Content: question},
	}

	return map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
		"header": map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
			"app_id": appid, //  应用appid，从开放平台控制台创建的应用中获取
		},
		"parameter": map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
			"chat": map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
				"domain":      "general", // 根据实际情况修改返回的数据结构和字段名
				"temperature": 1.0,       // 核采样阈值。用于决定结果随机性，取值越高随机性越强即相同的问题得到的不同答案的可能性越高
				"top_k":       1,         // 从k个候选中随机选择⼀个（⾮等概率）
				"max_tokens":  2048,      // 模型回答的tokens的最大长度
				"auditing":    "default", // 根据实际情况修改返回的数据结构和字段名
			},
		},
		"payload": map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
			"message": map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
				"text": messages, // 根据实际情况修改返回的数据结构和字段名
			},
		},
	}
}

// 创建鉴权url  apikey 即 hmac username
func assembleAuthUrl1(hosturl, apiKey, apiSecret string) string {
	ul, err := url.Parse(hosturl)
	if err != nil {
		fmt.Println("解析URL时错误", err)
		return ""
	}
	date := time.Now().UTC().Format(time.RFC1123)
	signString := []string{"host: " + ul.Host, "date: " + date, "GET " + ul.Path + " HTTP/1.1"}
	sgin := strings.Join(signString, "\n")
	sha := HmacWithShaTobase64("hmac-sha256", sgin, apiSecret)
	authUrl := fmt.Sprintf("hmac username=\"%s\", algorithm=\"%s\", headers=\"%s\", signature=\"%s\"", apiKey,
		"hmac-sha256", "host date request-line", sha)
	authorization := base64.StdEncoding.EncodeToString([]byte(authUrl))

	v := url.Values{}
	v.Add("host", ul.Host)
	v.Add("date", date)
	v.Add("authorization", authorization)
	return hosturl + "?" + v.Encode()
}

func HmacWithShaTobase64(algorithm, data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum((nil)))
}

func readResp(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("创建文件失败：%v", err)
	}
	return fmt.Sprintf("code=%d,body=%s", resp.StatusCode, string(b))
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// 将通道内的文字转字符串
func channelToString(results2 chan ScanResult) string {
	var sb strings.Builder

	for result := range results2 {
		sb.WriteString(fmt.Sprintf("%+v\n", result))
	}

	return sb.String()
}

// 定义一个方法返回处理后的结果给调用者
func GetAiText() []string {
	return aiText
}
