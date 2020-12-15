package allocator

import (
	"GPUMounter/pkg/config"
	"GPUMounter/pkg/device"
	"GPUMounter/pkg/util"
	"GPUMounter/pkg/util/gpu"
	"GPUMounter/pkg/util/gpu/collector"
	. "GPUMounter/pkg/util/log"
	"crypto/rand"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

type GPUAllocator struct {
	*collector.GPUCollector
}

func NewGPUAllocator() (*GPUAllocator, error) {
	Logger.Info("Creating gpu allocator")
	gpuAllocator := &GPUAllocator{}
	tmp, err := collector.NewGPUCollector()
	if err != nil {
		Logger.Error("Failed to init gpu collector")
		return nil, err
	}
	Logger.Info("Successfully created gpu collector")
	gpuAllocator.GPUCollector = tmp
	return gpuAllocator, nil
}

func (gpuAllocator *GPUAllocator) GetAvailableGPU(ownerPod *corev1.Pod, gpuNum int) ([]*device.NvidiaGPU, error) {
	clientset, err := config.GetClientSet()
	if err != nil {
		Logger.Error("Connect to k8s failed")
		return nil, errors.New(gpu.FailedCreated)
	}

	var slavePodNames []string
	for idx := 0; idx < gpuNum; idx++ {
		// try create a gpu pod on specify node
		slavePod := newGPUPod(ownerPod, 1)
		slavePod, err = clientset.CoreV1().Pods(slavePod.Namespace).Create(slavePod)
		if err != nil {
			Logger.Error("Failed to create GPU Slave Pod for Owner Pod: " + ownerPod.Name)
			return nil, errors.New(gpu.FailedCreated)
		}
		slavePodNames = append(slavePodNames, slavePod.Name)
		Logger.Info("Creating GPU Slave Pod: " + slavePod.Name + " for Owner Pod: " + ownerPod.Name)
	}

	ch := make(chan string)
	go checkState(slavePodNames, ch)
	switch <-ch {
	case gpu.InsufficientGPU:
		for _, slavePodName := range slavePodNames {
			err = clientset.CoreV1().Pods(ownerPod.Namespace).Delete(slavePodName, metav1.NewDeleteOptions(0))
			if err != nil {
				Logger.Error("Failed to recycle slave pod: ", slavePodName, " Namespace: ", ownerPod.Namespace)
			}
		}
		return nil, errors.New(gpu.InsufficientGPU)
	case gpu.FailedCreated:
		for _, slavePodName := range slavePodNames {
			err = clientset.CoreV1().Pods(ownerPod.Namespace).Delete(slavePodName, metav1.NewDeleteOptions(0))
			if err != nil {
				Logger.Error("Failed to recycle slave pod: ", slavePodName, " Namespace: ", ownerPod.Namespace)
			}
		}
		return nil, errors.New(gpu.FailedCreated)
	case gpu.SuccessfullyCreated:
		Logger.Info("Successfully create Slave Pod: %s, for Owner Pod: %s ", strings.Join(slavePodNames, ", "), ownerPod.Name)
		var availableGPUResource []*device.NvidiaGPU
		for _, slavePodName := range slavePodNames {
			gpuResources, err := gpuAllocator.GetPodGPUResources(slavePodName, ownerPod.Namespace)
			if err != nil {
				Logger.Error("Failed to get gpu resource for Slave Pod: ", slavePodName, " in Namespace: ", ownerPod.Namespace)
				return nil, errors.New(gpu.FailedCreated)
			}
			availableGPUResource = append(availableGPUResource, gpuResources...)
		}

		return availableGPUResource, nil
	}
	return nil, errors.New(gpu.FailedCreated)
}

func (gpuAllocator *GPUAllocator) GetRemoveGPU(ownerPod *corev1.Pod, uuids []string) ([]*device.NvidiaGPU, error) {

	gpuResources, err := gpuAllocator.GetPodGPUResources(ownerPod.Name, ownerPod.Namespace)
	if err != nil {
		Logger.Error("Failed to Get Pod: ", ownerPod.Name, " Namespace: ", ownerPod.Namespace, " GPU resources")
		Logger.Error(err)
		return nil, err
	}
	var removeGPUs []*device.NvidiaGPU
	for _, gpuDev := range gpuResources {
		// GPU Mounter can only unmount the gpu mounted by GPU Mounter
		// so the removed gpu should belong to slave pod
		if util.ContainString(uuids, gpuDev.UUID) && gpuDev.PodName != ownerPod.Name {
			removeGPUs = append(removeGPUs, gpuDev)
		}
	}
	// if exists unmatch gpu, return empty
	if len(uuids) != len(removeGPUs) {
		return []*device.NvidiaGPU{}, nil
	}

	return removeGPUs, nil
}

func newGPUPod(ownerPod *corev1.Pod, gpuNum int) *corev1.Pod {
	// generate random ID
	randBytes := make([]byte, 3)
	rand.Read(randBytes)
	randID := fmt.Sprintf("%x", randBytes)

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ownerPod.Name + "-slave-pod-" + randID,
			Namespace: ownerPod.Namespace,
			Labels: map[string]string{
				"app": "gpu-pool",
			},
			// set owner ref, so the slave pod will be auto removed if owner pod was removed
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					Kind:               "Pod",
					Name:               ownerPod.GetName(),
					UID:                ownerPod.GetUID(),
					BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
					Controller:         func(b bool) *bool { return &b }(true),
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "gpu-container",
					Image:   "alpine:latest",
					Command: []string{"/bin/sh"},
					Args:    []string{"-c", "while true; do echo this is a gpu pool container; sleep 10;done"},
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"nvidia.com/gpu": resource.MustParse(strconv.Itoa(gpuNum)),
						},
					},
				},
			},
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": ownerPod.Spec.NodeName,
			},
		},
		Status: corev1.PodStatus{},
	}
}

func checkState(names []string, ch chan string) {

	Logger.Info("Checking Pods: " + strings.Join(names, ", ") + " state")
	clientset, err := config.GetClientSet()
	if err != nil {
		Logger.Error("Connect to k8s failed")
		ch <- gpu.FailedCreated
	}

	for {
		flag := true
		for _, slavePodName := range names {
			pod, err := clientset.CoreV1().Pods("default").Get(slavePodName, metav1.GetOptions{})
			if err != nil {
				if k8s_errors.IsNotFound(err) {
					Logger.Info("Not Found....")
					continue
				} else {
					Logger.Error(err)
					ch <- gpu.FailedCreated
					return
				}
			}
			if pod.Status.Phase == "Running" {
				continue
			} else if !(len(pod.Status.Conditions) > 0) {
				flag = false
				Logger.Info("Pod: " + slavePodName + " creating")
				continue
			} else if pod.Status.Conditions[0].Reason == corev1.PodReasonUnschedulable {
				flag = false
				Logger.Info("No enough gpu for Pod: ", slavePodName)
				ch <- gpu.InsufficientGPU
				return
			} else {
				// pod is creating
				flag = false
			}
		}
		if flag {
			Logger.Info("Pods: " + strings.Join(names, ", ") + " are running")
			ch <- gpu.SuccessfullyCreated
			return
		}
	}
}
