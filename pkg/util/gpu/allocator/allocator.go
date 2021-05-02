package allocator

import (
	"GPUMounter/pkg/config"
	"GPUMounter/pkg/device"
	"GPUMounter/pkg/util"
	"GPUMounter/pkg/util/gpu"
	"GPUMounter/pkg/util/gpu/collector"
	. "GPUMounter/pkg/util/log"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (gpuAllocator *GPUAllocator) GetAvailableGPU(ownerPod *corev1.Pod, totalGpuNum int, gpuNumPerPod int) ([]*device.NvidiaGPU, error) {
	clientset, err := config.GetClientSet()
	if err != nil {
		Logger.Error(err)
		Logger.Error("Connect to k8s failed")
		return nil, errors.New(gpu.FailedCreated)
	}

	var slavePodNames []string
	for idx := 0; idx < totalGpuNum/gpuNumPerPod; idx++ {
		// try create a gpu pod on specify node
		slavePod := newGPUSlavePod(ownerPod, gpuNumPerPod)
		slavePod, err = clientset.CoreV1().Pods(slavePod.Namespace).Create(context.TODO(), slavePod, metav1.CreateOptions{})
		if err != nil {
			Logger.Error(err)
			Logger.Error("Failed to create GPU Slave Pod for Owner Pod: " + ownerPod.Name)
			return nil, errors.New(gpu.FailedCreated)
		}
		slavePodNames = append(slavePodNames, slavePod.Name)
		Logger.Info("Creating GPU Slave Pod: " + slavePod.Name + " for Owner Pod: " + ownerPod.Name)
	}

	ch := make(chan string)
	go checkCreateState(slavePodNames, ch)
	switch <-ch {
	case gpu.InsufficientGPU:
		for _, slavePodName := range slavePodNames {
			err = clientset.CoreV1().Pods(gpu.GPUPoolNamespace).Delete(context.TODO(), slavePodName, *metav1.NewDeleteOptions(0))
			if err != nil {
				Logger.Error(err)
				Logger.Error("Failed to recycle slave pod: ", slavePodName, " Namespace: ", gpu.GPUPoolNamespace)
			}
		}
		return nil, errors.New(gpu.InsufficientGPU)
	case gpu.FailedCreated:
		for _, slavePodName := range slavePodNames {
			err = clientset.CoreV1().Pods(gpu.GPUPoolNamespace).Delete(context.TODO(), slavePodName, *metav1.NewDeleteOptions(0))
			if err != nil {
				Logger.Error(err)
				Logger.Error("Failed to recycle slave pod: ", slavePodName, " Namespace: ", gpu.GPUPoolNamespace)
			}
		}
		return nil, errors.New(gpu.FailedCreated)
	case gpu.SuccessfullyCreated:
		Logger.Info("Successfully create Slave Pod: %s, for Owner Pod: %s ", strings.Join(slavePodNames, ", "), ownerPod.Name)
		var availableGPUResource []*device.NvidiaGPU
		for _, slavePodName := range slavePodNames {
			gpuResources, err := gpuAllocator.GetPodGPUResources(slavePodName, gpu.GPUPoolNamespace)
			if err != nil {
				Logger.Error(err)
				Logger.Error("Failed to get gpu resource for Slave Pod: ", slavePodName, " in Namespace: ", gpu.GPUPoolNamespace)
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
		Logger.Error(err)
		Logger.Error("Failed to Get Pod: ", ownerPod.Name, " Namespace: ", ownerPod.Namespace, " GPU resources")
		return nil, err
	}

	var removeGPUs []*device.NvidiaGPU
	mountType := gpuAllocator.GetMountType(ownerPod)
	for _, gpuDev := range gpuResources {
		// GPU Mounter can only unmount the gpu mounted by GPU Mounter
		// so the removed gpu should belong to slave pod
		// if entire mount pod, remove all gpu
		if (mountType == gpu.EntireMount || util.ContainString(uuids, gpuDev.UUID)) && gpuDev.PodName != ownerPod.Name {
			removeGPUs = append(removeGPUs, gpuDev)
		}
	}
	// if exists unmatch gpu, return empty
	if len(uuids) != len(removeGPUs) {
		return []*device.NvidiaGPU{}, nil
	}

	return removeGPUs, nil
}

func (gpuAllocator *GPUAllocator) DeleteSlavePods(slavePodNames []string) error {
	Logger.Info("Deleting slave pods: ", strings.Join(slavePodNames, ", "))
	clientset, err := config.GetClientSet()
	if err != nil {
		Logger.Error("Connect to k8s failed")
		return err
	}
	for _, slavePodName := range slavePodNames {
		err = clientset.CoreV1().Pods(gpu.GPUPoolNamespace).Delete(context.TODO(), slavePodName, metav1.DeleteOptions{})
		if err != nil {
			Logger.Error("Failed to delete Slave Pod: ", slavePodName)
			return err
		}
	}

	ch := make(chan string)
	go checkDeleteState(slavePodNames, ch)

	switch <-ch {
	case gpu.FailedDeleted:
		Logger.Error("Failed to delete slave pods")
		return errors.New("Failed to delete slave pods ")
	case gpu.SuccessfullyDeleted:
		Logger.Info("Successfully delete slave pods")
		return nil
	}
	return errors.New("Unkown status from checking goroutine ")

}

func (gpuAllocator *GPUAllocator) GetMountType(pod *corev1.Pod) gpu.MountType {
	Logger.Infof("Get pod %s/%s mount type", pod.Namespace, pod.Name)
	gpuResources, err := gpuAllocator.GetPodGPUResources(pod.Name, pod.Namespace)
	if err != nil {
		Logger.Error(err)
		Logger.Error("Failed to get Pod: ", pod.Name, " Namespace: ", pod.Namespace, " mount type")
		return gpu.UnknownMount
	}

	if len(gpuResources) == 0 {
		return gpu.NoMount
	}

	slavePodNames := make(map[string]interface{}, 0)
	gpuNum := 0
	for _, gpuDev := range gpuResources {
		if gpuDev.PodName != pod.Name {
			slavePodNames[gpuDev.PodName] = struct{}{}
		}
		gpuNum++
	}

	// entire mount pod has less slave pod than its gpu num
	// TODO: here we regard a mount as entire mount if pod's gpu num less than slave pods, any better way?
	if len(slavePodNames) < gpuNum {
		return gpu.EntireMount
	} else {
		return gpu.SingleMount
	}
}

func newGPUSlavePod(ownerPod *corev1.Pod, gpuNum int) *corev1.Pod {
	// generate random ID
	randBytes := make([]byte, 3)
	rand.Read(randBytes)
	randID := fmt.Sprintf("%x", randBytes)

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ownerPod.Name + "-slave-pod-" + randID,
			Namespace: gpu.GPUPoolNamespace,
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

func checkCreateState(podNames []string, ch chan string) {

	Logger.Info("Checking Pods: " + strings.Join(podNames, ", ") + " state")
	clientset, err := config.GetClientSet()
	if err != nil {
		Logger.Error(err)
		Logger.Error("Connect to k8s failed")
		ch <- gpu.FailedCreated
	}

	for {
		flag := true
		for _, slavePodName := range podNames {
			pod, err := clientset.CoreV1().Pods(gpu.GPUPoolNamespace).Get(context.TODO(), slavePodName, metav1.GetOptions{})
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
			Logger.Info("Pods: " + strings.Join(podNames, ", ") + " are running")
			ch <- gpu.SuccessfullyCreated
			return
		}
	}
}

func checkDeleteState(podNames []string, ch chan string) {

	Logger.Info("Checking Pods: " + strings.Join(podNames, ", ") + " state")
	clientset, err := config.GetClientSet()
	if err != nil {
		Logger.Error(err)
		Logger.Error("Connect to k8s failed")
		ch <- gpu.FailedDeleted
		return
	}

	for {
		flag := true
		for _, slavePodName := range podNames {
			_, err := clientset.CoreV1().Pods(gpu.GPUPoolNamespace).Get(context.TODO(), slavePodName, metav1.GetOptions{})
			if err != nil {
				if k8s_errors.IsNotFound(err) {
					// this slavePod has been deleted
					continue
				} else {
					Logger.Error(err)
					ch <- gpu.FailedDeleted
					return
				}
			}
			flag = false
		}
		if flag {
			Logger.Info("Pods: " + strings.Join(podNames, ", ") + " deleted successfully")
			ch <- gpu.SuccessfullyDeleted
			return
		}
	}
}
