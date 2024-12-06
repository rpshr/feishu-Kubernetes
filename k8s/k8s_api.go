package k8s

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	sendmsg "testapi/sedmsg"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

// 配置文件路径和命名空间
// var kubeconfigPath = `D:`
// var namespace = "test"
const (
	kubeconfigPath = `/xxxx/xxx/.config`
	namespace      = "xxxx"
	// 定义某些项目不走飞书审批自动
	xxxx      = "xxxxxx"
	mydefault = "xxxxx"
)

// 使用正则表达式判断 jobName 是否包含任何一个常量
func regexpString(jobName string) bool {
	// 对常量进行转义
	escapedConstant := regexp.QuoteMeta(xxxx)
	//(?:^|[-])crm(?:[-]|$)
	// 构建正则表达式模式
	pattern := fmt.Sprintf(`(?:^|[-])%s(?:[-]|$)`, escapedConstant)
	re := regexp.MustCompile(pattern)
	// 检查 jobName 是否匹配
	if match := re.FindString(jobName); match != "" {
		return false
	}
	return true
}

// FeishuDeployments 更新指定 Deployment 的镜像并检查 Pod 状态
func FeishuDeployments(jobName, versionNumber string) error {
	if !regexpString(jobName) {
		sendmsg.SendInteractiveMsg("智慧门店不使用这个审批流程", jobName, "red")
		return fmt.Errorf("xxxxx项目%s不使用这个审批流程", jobName)
	}

	// 从 kubeconfig 文件创建 Kubernetes 配置
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create config: %v", err)
	}

	// 创建 Kubernetes 客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	// 获取指定的 Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), jobName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get deployment %s: %v", jobName, err)
		return err
	}

	// 查找并更新指定容器的镜像
	found := false
	for i := range deployment.Spec.Template.Spec.Containers {
		if deployment.Spec.Template.Spec.Containers[i].Name == jobName {
			imageParts := strings.Split(deployment.Spec.Template.Spec.Containers[i].Image, ":")
			if len(imageParts) < 2 {
				return fmt.Errorf("invalid image name for container %s: %s", jobName, deployment.Spec.Template.Spec.Containers[i].Image)
			}
			deployment.Spec.Template.Spec.Containers[i].Image = fmt.Sprintf("%s:%s", imageParts[0], versionNumber)
			found = true
			break
		}
	}

	if !found {
		klog.Errorf("Container with name '%s' not found in deployment %s", jobName, jobName)
		err := fmt.Sprintf("Container with name '%s' not found in deployment %s", jobName, jobName)
		sendmsg.SendInteractiveMsg(err, jobName, "blue")
		return fmt.Errorf("container with name '%s' not found in deployment %s", jobName, jobName)
	}

	// 更新 Deployment
	_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Failed to update deployment %s: %v", jobName, err)
		errors := fmt.Sprintf("Failed to update deployment %s: %v", jobName, err)
		sendmsg.SendInteractiveMsg(errors, jobName, "blue")
		return err
	}

	klog.Infof("Deployment %s updated successfully. New image: %s", jobName, versionNumber)
	successfully := fmt.Sprintf("Deployment %s updated successfully. New image: %s", jobName, versionNumber)
	sendmsg.SendInteractiveMsg(successfully, jobName, "green")

	// 检查 Pod 状态
	err = CheckDeploymentPodStatusfat(clientset, jobName, versionNumber)
	if err != nil {
		klog.Errorf("Failed to check deployment pod status: %v", err)
		errs := fmt.Sprintf("Failed to check deployment pod status: %v", err)
		sendmsg.SendInteractiveMsg(errs, jobName, "blue")
		return err
	}
	sucmsg := fmt.Sprintf("successfully to check %s pod status: ok", jobName)
	sendmsg.SendInteractiveMsg(sucmsg, jobName, "green")
	return nil
}

// ExtractImageAndVersion 使用正则表达式提取镜像名称和版本
func ExtractImageAndVersion(jobName, versionNumber string) (string, string, error) {
	// 构建完整的镜像路径
	registry := "tastien-registry-vpc.cn-shanghai.cr.aliyuncs.com"
	repository := "tst-uat"

	// 去掉 -gray-level 后缀
	imageName := strings.TrimSuffix(jobName, "-gray-level")
	fullImageName := fmt.Sprintf("%s/%s/%s", registry, repository, imageName)
	latestImage := fmt.Sprintf("%s:%s", fullImageName, versionNumber)

	return latestImage, versionNumber, nil
}

// ExtractImageAndVersion 使用正则表达式提取镜像名称和版本
func ExtractImageAndVersionfat(jobName, versionNumber string) (string, string, error) {
	imageString := fmt.Sprintf("%s:%s", jobName, versionNumber)
	re := regexp.MustCompile(`^([a-zA-Z0-9-]+)(?:-gray-level)*:(.*)$`)
	matches := re.FindStringSubmatch(imageString)
	if matches == nil || len(matches) != 3 {
		return "", "", fmt.Errorf("invalid image string format: %s", imageString)
	}
	imageName := matches[1]
	version := matches[2]
	imageName = strings.TrimSuffix(imageName, "-gray-level")
	return imageName, version, nil
}

// CheckDeploymentPodStatus 检查 Deployment 的 Pod 状态，确保至少有一个 Pod 使用最新镜像且处于运行和就绪状态
func CheckDeploymentPodStatusfat(clientset *kubernetes.Clientset, jobName, versionNumber string) error {
	imageName, version, err := ExtractImageAndVersionfat(jobName, versionNumber)
	if err != nil {
		return err
	}

	latestImage := fmt.Sprintf("tastien-registry-vpc.cn-shanghai.cr.aliyuncs.com/%s/%s:%s", namespace, imageName, version)
	klog.Infof("Latest Image: %s", latestImage)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for at least one pod with the latest image to be running and ready: %s", latestImage)
		default:
			// 获取 Deployment
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), jobName, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return fmt.Errorf("deployment %s not found in namespace %s", jobName, namespace)
				}
				return fmt.Errorf("failed to get deployment %s: %v", jobName, err)
			}

			// 构建 Label Selector
			labels := deployment.Spec.Template.ObjectMeta.Labels
			labelSelector := ""
			for key, value := range labels {
				if labelSelector != "" {
					labelSelector += ","
				}
				labelSelector += fmt.Sprintf("%s=%s", key, value)
			}
			klog.Infof("Label Selector: %s", labelSelector)

			// 列出所有匹配 Label Selector 的 Pod
			podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				return fmt.Errorf("failed to list pods: %v", err)
			} else if podList == nil || len(podList.Items) == 0 {
				time.Sleep(15 * time.Second)
				if len(podList.Items) == 0 {
					klog.Warningf("The current number of Pods for the %s is 0", jobName)
					//klog.Errorf("The current number of Pods for the %s is 0", jobName)
					//continue
					return fmt.Errorf("the current number of Pods for the %s is 0", jobName)
				}
			}

			// 检查每个 Pod 的状态
			for _, pod := range podList.Items {
				podName := pod.ObjectMeta.Name
				podStatus := pod.Status.Phase
				containerStatuses := pod.Status.ContainerStatuses

				klog.Infof("Checking Pod: %s, Status: %s", podName, podStatus)

				if podStatus != corev1.PodRunning {
					klog.Warningf("Pod %s is not in Running state, current state: %s", podName, podStatus)
					continue
				}

				// 检查每个容器的状态
				for _, containerStatus := range containerStatuses {
					klog.Infof("Checking container %s in Pod %s, Image: %s, Ready: %v", containerStatus.Name, podName, containerStatus.Image, containerStatus.Ready)
					if containerStatus.Image == latestImage && containerStatus.State.Running != nil && containerStatus.Ready {
						klog.Infof("Found Pod: %s with the latest image running and ready: %s", podName, latestImage)
						return nil
					}
				}
			}

			klog.Infof("No pod with the latest image is running and ready yet. Retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
		}
	}
}

// CheckDeploymentPodStatus 检查 Deployment 的 Pod 状态，确保至少有一个 Pod 使用最新镜像且处于运行和就绪状态
func CheckDeploymentPodStatus(clientset *kubernetes.Clientset, jobName, versionNumber string) error {
	latestImage, _, err := ExtractImageAndVersion(jobName, versionNumber)
	if err != nil {
		return err
	}

	klog.Infof("Latest Image: %s", latestImage)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for at least one pod with the latest image to be running and ready: %s", latestImage)
		default:
			// 获取 Deployment
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), jobName, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return fmt.Errorf("deployment %s not found in namespace %s", jobName, namespace)
				}
				return fmt.Errorf("failed to get deployment %s: %v", jobName, err)
			}

			// 构建 Label Selector
			labels := deployment.Spec.Template.ObjectMeta.Labels
			labelSelector := ""
			for key, value := range labels {
				if labelSelector != "" {
					labelSelector += ","
				}
				labelSelector += fmt.Sprintf("%s=%s", key, value)
			}
			klog.Infof("Label Selector: %s", labelSelector)

			// 列出所有匹配 Label Selector 的 Pod
			podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				return fmt.Errorf("failed to list pods: %v", err)
			} else if podList == nil || len(podList.Items) == 0 {
				klog.Warningf("The current number of Pods for the %s is 0", jobName)
				continue
			}

			// 检查每个 Pod 的状态
			for _, pod := range podList.Items {
				podName := pod.ObjectMeta.Name
				podStatus := pod.Status.Phase
				containerStatuses := pod.Status.ContainerStatuses

				klog.Infof("Checking Pod: %s, Status: %s", podName, podStatus)

				if podStatus != corev1.PodRunning {
					klog.Warningf("Pod %s is not in Running state, current state: %s", podName, podStatus)
					continue
				}

				// 检查每个容器的状态
				for _, containerStatus := range containerStatuses {
					klog.Infof("Checking container %s in Pod %s, Image: %s, Ready: %v", containerStatus.Name, podName, containerStatus.Image, containerStatus.Ready)
					if containerStatus.Image == latestImage && containerStatus.State.Running != nil && containerStatus.Ready {
						klog.Infof("Found Pod: %s with the latest image running and ready: %s", podName, latestImage)
						return nil
					}
				}
			}

			klog.Infof("No pod with the latest image is running and ready yet. Retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
		}
	}
}
