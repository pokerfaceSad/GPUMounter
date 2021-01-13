package allocator

import (
	"GPUMounter/pkg/config"
	. "GPUMounter/pkg/util/log"
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestGetAvailableGPU(t *testing.T) {
	InitLogger(".", "log")
	defer Logger.Sync()

	gpuAllocator, err := NewGPUAllocator()
	if err != nil {
		Logger.Error("Failed to init gpu allocator")
		panic(err)
	}
	clientset, err := config.GetClientSet()
	if err != nil {
		Logger.Error("Connect to k8s failed")
		panic(err)
	}
	pod, err := clientset.CoreV1().Pods("default").Get(context.TODO(), "gpu-pod2", metav1.GetOptions{})
	if err != nil {
		Logger.Error("get pod " + pod.Name + " failed")
		panic(err)
	}
	gpuResources, err := gpuAllocator.GetAvailableGPU(pod, 2)
	if err != nil {
		panic(err)
	}
	for _, gpuDev := range gpuResources {
		Logger.Info(gpuDev)
	}
}
