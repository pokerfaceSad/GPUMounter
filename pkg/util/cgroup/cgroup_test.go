package cgroup

import (
	"GPUMounter/pkg/config"
	"GPUMounter/pkg/util/log"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/exec"
	"strings"
	"testing"
)

func TestGetCgroupName(t *testing.T) {
	clientset, err := config.GetClientSet()
	if err != nil {
		panic(err)
	}
	pod, err := clientset.CoreV1().Pods("default").Get("gpu-pod", metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	containerID := pod.Status.ContainerStatuses[0].ContainerID
	containerID = strings.Replace(containerID, "docker://", "", 1)
	fmt.Println(containerID)
	cgroupName, err := GetCgroupName("cgroupfs", pod, containerID)
	if err != nil {
		panic(err)
	}
	fmt.Println(cgroupName)

	deviceCgroupPath := "/sys/fs/cgroup/devices" + cgroupName
	out, err := exec.Command("sh", "-c", "echo 'c 195:0 rwm' > "+deviceCgroupPath+"/devices.allow").CombinedOutput()
	if err != nil {
		fmt.Println(string(out[:]))
		panic(err)
	}
	fmt.Println(string(out[:]))
}

func TestGetCgroupPID(t *testing.T) {
	log.InitLogger("./", "log")
	defer log.Logger.Sync()
	//clientset, err := config.GetClientSet()
	//if err != nil {
	//	panic(err)
	//}
	//pod, err := clientset.CoreV1().Pods("default").Get("tomcat-deployment-7dc99cbdbb-9s7sk", metav1.GetOptions{})
	//if err != nil {
	//	panic(err)
	//}
	//containerID := pod.Status.ContainerStatuses[0].ContainerID
	//containerID = strings.Replace(containerID, "docker://", "", 1)
	//fmt.Println(containerID)
	//cgroupName, err := GetCgroupName("cgroupfs", pod, containerID)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(cgroupName)
	PIDs, err := GetCgroupPIDs("/docker/42022f8a20343c298ef0bff029b3871b470b9c357ca3723e347f6cc56dc72389")
	if err != nil {
		panic(err)
	}
	for idx, pid := range PIDs {
		fmt.Println(idx, " : "+pid+"-")
	}
}
