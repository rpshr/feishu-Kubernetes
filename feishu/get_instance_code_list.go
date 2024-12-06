package feishu

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testapi/k8s"
	myredis "testapi/redis"
	"time"
)

// InstanceListResponse 结构体定义
type InstanceListResponse struct {
	Code int `json:"code"`
	Data struct {
		HasMore          bool     `json:"has_more"`
		InstanceCodeList []string `json:"instance_code_list"`
		PageToken        string   `json:"page_token"`
	} `json:"data"`
	Msg string `json:"msg"`
}

// GetInstancekubeCodeList 函数
func GetInstancekubeCodeList() {
	// 获取当前日期的开始和结束时间
	now := time.Now()
	year, month, day := now.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	endOfDay := time.Date(year, month, day, 23, 59, 59, 0, now.Location())

	startTimestamp := startOfDay.UnixNano() / int64(time.Millisecond)
	endTimestamp := endOfDay.UnixNano() / int64(time.Millisecond)

	// 构建请求URL
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/approval/v4/instances?approval_code=xxxxxxxxxxxxxxx&end_time=%d&page_size=100&start_time=%d", endTimestamp, startTimestamp)

	// 创建HTTP GET请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		return
	}

	// 获取租户访问令牌
	tenantAccessToken, err := GetTenantAccessToken()
	if err != nil {
		log.Printf("获取租户访问令牌失败: %v", err)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tenantAccessToken))

	// 发送HTTP请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("发送请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return
	}

	// 打印完整的响应体
	log.Printf("完整的响应体: %s", string(body))

	// 解析响应
	var response InstanceListResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Printf("解析响应失败: %v", err)
		return
	}

	// 检查响应状态码
	if response.Code != 0 { // 假设0表示成功
		log.Printf("请求失败: %s", response.Msg)
		return
	}

	// 处理每个审批实例
	for _, instanceCode := range response.Data.InstanceCodeList {
		fmt.Println("====================================================")
		fmt.Println("!!!!!!!我的data", instanceCode, "test--------------")

		// 检查Redis中是否已经处理过该实例
		exists, err := myredis.CreateRedisInstance("get", instanceCode)
		if err != nil {
			log.Printf("检查Redis失败: %v", err)
			continue
		}

		// 确保 exists 是一个布尔值
		existsBool, ok := exists.(bool)
		if !ok {
			log.Printf("exists 不是布尔类型: %T", exists)
			continue
		}

		if existsBool {
			fmt.Println("这个审批已经处理过了:", instanceCode)
			log.Printf("审批单 %s 已经处理过，跳过处理。", instanceCode)
			continue
		}

		// 获取项目信息
		projectInfo, err := GetProjectInfo(instanceCode, true)
		if err != nil {
			log.Printf("获取项目信息失败: %v", err)
			continue
		}

		// 从 projectInfo 中获取内层映射
		info, ok := projectInfo[instanceCode]
		if !ok {
			log.Printf("项目信息中没有找到实例代码: %s", instanceCode)
			continue
		}

		// 打印项目信息
		fmt.Println("====================")
		fmt.Println("下面打印的是审批获取到的所有信息")
		fmt.Printf("JobNameAndVersionNumber: %s, Status: %s, 审批单实例ID: %s\n",
			info["JobNameAndVersionNumber"],
			info["status"],
			instanceCode)
		fmt.Println("====================")

		// 获取 status 字段
		status := info["status"]

		// 根据状态处理
		switch status {
		case "APPROVED":
			fmt.Println("审批单已经通过了下面开始执行发版本程序")
			fmt.Printf("项目名称版本号: %s,  审批单状态: %s\n",
				info["JobNameAndVersionNumber"],
				status)

			// 解析JobNameAndVersionNumber
			jobNamesAndVersions := strings.Split(info["JobNameAndVersionNumber"], "\n")
			for _, jnv := range jobNamesAndVersions {
				jnv = strings.TrimSpace(jnv)
				if jnv == "" {
					continue
				}

				// 使用strings.Fields处理一个或多个空格作为分隔符
				parts := strings.Fields(jnv)
				if len(parts) != 2 {
					log.Printf("无法解析JobNameAndVersionNumber: %s", jnv)
					continue
				}
				jobName := parts[0]
				versionNumber := parts[1]

				// 执行Kubernetes部署
				err := k8s.FeishuDeployments(jobName, versionNumber)
				if err != nil {
					log.Printf("执行Kubernetes部署失败: %v", err)
					continue
				}
			}

			// 设置Redis键
			_, err = myredis.CreateRedisInstance("set", instanceCode, "123")
			if err != nil {
				log.Printf("设置Redis key失败: %v", err)
			}
		case "PENDING":
			fmt.Println("单子正在审批中，请耐心等待")
		case "REJECTED":
			fmt.Println("发版被拒绝, 请找管理员确认原因")
			// 设置Redis键
			_, err = myredis.CreateRedisInstance("set", instanceCode, "123")
			if err != nil {
				log.Printf("设置Redis key失败: %v", err)
			}
		default:
			log.Printf("未知的审批状态: %s", status)
		}
	}
}
