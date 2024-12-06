package feishu

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// JobData 结构体定义
type JobData struct {
	JobName       string `json:"JobName"`
	VersionNumber string `json:"VersionNumber"`
}

// FormField 结构体定义
type FormField struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// JobDatak8s 结构体定义
type JobDatak8s struct {
	JobNameAndVersionNumber string `json:"JobNameAndVersionNumber"`
}

// FormFieldk8s 结构体定义
type FormFieldk8s struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// GetProjectInfo 函数
func GetProjectInfo(instanceCode string, useK8sFormat bool) (map[string]map[string]string, error) {
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/approval/v4/instances/%s", instanceCode)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 使用GetTenantAccessToken()函数获取tenantAccessToken
	tenantAccessToken, err := GetTenantAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取租户访问令牌失败: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tenantAccessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 打印完整的响应体
	log.Printf("完整的响应体: %s", string(body))

	// 解析整个响应
	var response struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查响应状态码
	if response.Code != 0 { // 假设0表示成功
		return nil, fmt.Errorf("请求失败: %s", response.Msg)
	}

	// 获取数据部分
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("响应数据格式错误: 'data' 字段不是预期的 map 类型")
	}
	fmt.Println("审批单信息：", data)

	// 获取当前时间
	now := time.Now().UTC()
	nowTimestamp := now.UnixNano() / int64(time.Millisecond)

	// 初始化结果
	unexpiredResults := make(map[string]map[string]string)

	// 直接处理单个审批单的信息
	// 获取审批单的结束时间
	var endTime int64
	if endTimeInterface, ok := data["end_time"]; ok {
		switch v := endTimeInterface.(type) {
		case float64:
			endTime = int64(v * 1000) // 假设 end_time 是秒级时间戳
		case int64:
			endTime = v
		case string:
			endTimeInt, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				log.Printf("解析 end_time 字段失败: %v", err)
				// 如果类型未知，默认继续处理
				endTime = 0
			} else {
				endTime = endTimeInt
			}
		default:
			log.Printf("响应数据格式警告: 'end_time' 字段类型未知，将忽略此字段并继续处理")
			// 如果类型未知，默认继续处理
			endTime = 0
		}
	} else {
		// 如果 end_time 不存在，认为审批单仍在审批中
		endTime = 0
	}

	// 比较结束时间和当前时间，加上 1 小时的缓冲时间
	bufferedEndTime := endTime + int64(60*1000) // 1 小时的缓冲时间
	if bufferedEndTime > 0 && bufferedEndTime < nowTimestamp {
		// 审批单已过期，跳过
		return unexpiredResults, nil
	}

	// 审批单未过期
	unexpiredResult := make(map[string]string)
	// 获取状态
	status, ok := data["status"].(string)
	if !ok {
		return nil, fmt.Errorf("响应数据格式错误: 'status' 字段不是预期的 string 类型")
	}
	unexpiredResult["status"] = status

	// 获取表单数据
	formStr, ok := data["form"].(string)
	if !ok {
		return nil, fmt.Errorf("响应数据格式错误: 'form' 字段不是预期的 string 类型")
	}

	// 根据useK8sFormat选择解析方式
	var form []FormField
	if useK8sFormat {
		var formK8s []FormFieldk8s
		if err := json.Unmarshal([]byte(formStr), &formK8s); err != nil {
			return nil, fmt.Errorf("解析form字段失败: %w", err)
		}
		form = convertToFormFieldSlice(formK8s)
	} else {
		if err := json.Unmarshal([]byte(formStr), &form); err != nil {
			return nil, fmt.Errorf("解析form字段失败: %w", err)
		}
	}

	// 提取表单字段
	for _, field := range form {
		unexpiredResult[field.Name] = field.Value
	}

	// 添加到未过期结果中
	instanceID, ok := data["instance_code"].(string)
	if !ok {
		return nil, fmt.Errorf("响应数据格式错误: 'instance_code' 字段不是预期的 string 类型")
	}
	unexpiredResults[instanceID] = unexpiredResult

	// 返回结果
	return unexpiredResults, nil
}

// convertToFormFieldSlice 将FormFieldk8s类型的切片转换为FormField类型的切片
func convertToFormFieldSlice(formK8s []FormFieldk8s) []FormField {
	result := make([]FormField, len(formK8s))
	for i, v := range formK8s {
		result[i] = FormField{
			ID:    v.ID,
			Name:  v.Name,
			Value: v.Value,
		}
	}
	return result
}
