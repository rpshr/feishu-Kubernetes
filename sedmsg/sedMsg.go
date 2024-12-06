package sendmsg

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// 定义服务名称常量
const (
	mydefault = "mysdefault"
)

// 定义 API URLs 常量
const (
	mydefaultapi = "https://open.feishu.cn/open-apis/bot/v2/hookxxxxxxx"
)

// 定义一个不可变的 map
var APIMap = map[string]string{
	mydefault: mydefaultapi,
}

// 使用正则表达式判断 jobName 是否包含任何一个常量
func regexpString(jobName string) string {
	for k := range APIMap {
		// 对常量进行转义
		escapedConstant := regexp.QuoteMeta(k)
		//(?:^|[-])crm(?:[-]|$)
		// 构建正则表达式模式
		pattern := fmt.Sprintf(`(?:^|[-])%s(?:[-]|$)`, escapedConstant)
		re := regexp.MustCompile(pattern)
		// 检查 jobName 是否匹配
		if match := re.FindString(jobName); match != "" {
			// 如果匹配，返回对应的 API URL
			return APIMap[k]
		}
	}
	// 如果没有匹配到任何常量，返回默认的 API URL
	return mydefaultapi
}

func sendFeishuMsg(apiUrl, contentType, data string) error {
	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP response status: %s", resp.Status)
	}
	return nil
}

func SendInteractiveMsg(message, jobName, colors string) {
	apiUrl := regexpString(jobName)
	contentType := "application/json"

	currentTime := time.Now().Format("2006-01-02 15:04:05")

	cardMessage := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"config": map[string]bool{
				"wide_screen_mode": true,
			},
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"content": "发版通知",
					"tag":     "plain_text",
				},
				//"template": "blue",
				//green
				//red
				"template": colors,
			},
			"elements": []interface{}{
				map[string]interface{}{
					"tag": "div",
					"fields": []interface{}{
						//map[string]interface{}{"is_short": true, "text": map[string]interface{}{"content": "**告警级别**", "tag": "lark_md"}},
						//map[string]interface{}{"is_short": false, "text": map[string]interface{}{"content": alertLevel, "tag": "lark_md"}},
						//map[string]interface{}{"is_short": true, "text": map[string]interface{}{"content": "**资源**", "tag": "lark_md"}},
						//map[string]interface{}{"is_short": false, "text": map[string]interface{}{"content": resource, "tag": "lark_md"}},
						map[string]interface{}{"is_short": true, "text": map[string]interface{}{"content": "**时间**", "tag": "lark_md"}},
						map[string]interface{}{"is_short": false, "text": map[string]interface{}{"content": currentTime, "tag": "lark_md"}},
						map[string]interface{}{"is_short": true, "text": map[string]interface{}{"content": "**详情**", "tag": "lark_md"}},
						map[string]interface{}{"is_short": false, "text": map[string]interface{}{"content": message + "\n", "tag": "lark_md"}},
					},
				},
			},
		},
	}

	sendData, err := json.Marshal(cardMessage)
	if err != nil {
		fmt.Printf("json.Marshal failed, err: %v\n", err)
		return
	}

	if err := sendFeishuMsg(apiUrl, contentType, string(sendData)); err != nil {
		fmt.Printf("post failed, err: %v\n", err)
	} else {
		fmt.Println("告警已成功发送")
	}
}
