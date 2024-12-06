package jenkins

import (
	"context"
	"log"
	"net/http"

	"github.com/bndr/gojenkins"
)

// BuildRequest 结构体定义
type BuildRequest struct {
	JobName            string `json:"JobName"`
	GitlabSourceBranch string `json:"GitlabSourceBranch"`
	ChangeType         string `json:"ChangeType"`
}

// BuildHandler 函数
func BuildHandler(jobName, changeType, gitlabSourceBranch string) error {
	// 创建 HTTP 客户端
	httpClient := &http.Client{}
	// 创建一个空的上下文对象
	ctx := context.Background()

	// 创建 Jenkins 实例

	jenkins, err := gojenkins.CreateJenkins(httpClient, "http://xx.xx.xx.xx:xxx/", "xxx", "xxx").Init(ctx)
	if err != nil {
		log.Printf("无法初始化 Jenkins: %v", err)
		return err
	}

	// 获取指定 Job 的信息
	job, err := jenkins.GetJob(ctx, jobName)
	if err != nil {
		log.Printf("无法获取 Job '%s': %v", jobName, err)
		return err
	}

	// 为指定 Job 和分支构建
	params := map[string]string{
		"CHANGE_TYPE":        changeType,
		"gitlabSourceBranch": gitlabSourceBranch,
	}

	build, err := job.InvokeSimple(ctx, params)
	if err != nil {
		log.Printf("无法启动 Job '%s' 的构建: %v", jobName, err)
		return err
	}

	log.Printf("Build %d for job '%s' with branch '%s' is in queue", build, jobName, gitlabSourceBranch)
	return nil
}
