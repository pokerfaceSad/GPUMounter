package collector

import (
	"GPUMounter/pkg/device"
	"GPUMounter/pkg/util/gpu"
	"GPUMounter/pkg/util/gpu/collector/nvml"
	. "GPUMounter/pkg/util/log"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	podresourcesapi "k8s.io/kubernetes/pkg/kubelet/apis/podresources/v1alpha1"
)

type GPUCollector struct {
	GPUList []*device.NvidiaGPU
}

func NewGPUCollector() (*GPUCollector, error) {
	Logger.Info("Creating gpu collector")
	gpuCollector := &GPUCollector{}
	if err := gpuCollector.GetGPUInfo(); err != nil {
		Logger.Error("Failed to init gpu collector")
		return nil, err
	}

	err := gpuCollector.UpdateGPUStatus()
	if err != nil {
		Logger.Error("Failed to update gpu status")
		return nil, err
	}
	Logger.Info("Successfully update gpu status")
	return gpuCollector, nil
}

func (gpuCollector *GPUCollector) GetGPUInfo() error {

	Logger.Info("Start get gpu info")
	if err := nvml.Init(); err != nil {
		Logger.Error("nvml error: %+v", err)
		return err
	}
	defer nvml.Shutdown()

	num, err := nvml.GetDeviceCount()
	if err != nil {
		Logger.Error("Failed to get GPU num")
	} else {
		Logger.Info("GPU Num: ", num)
	}

	for i := uint(0); i < num; i++ {
		dev, err := nvml.DeviceGetHandleByIndex(i)
		if err != nil {
			Logger.Error("Failed to get GPU ", i)
			return err
		}

		minorNum, err := dev.DeviceGetMinorNumber()
		if err != nil {
			Logger.Error("Failed to get minor number of GPU ", i)
			return err
		}

		uuid, err := dev.DeviceGetUUID()
		if err != nil {
			Logger.Error("Failed to get uuid of GPU ", i)
			return err
		}

		gpuDev := device.New(int(minorNum), uuid)
		gpuCollector.GPUList = append(gpuCollector.GPUList, gpuDev)
	}
	return nil
}

func (gpuCollector *GPUCollector) GetGPUByUUID(uuid string) (*device.NvidiaGPU, error) {
	for _, gpuDev := range gpuCollector.GPUList {
		if gpuDev.UUID == uuid {
			return gpuDev, nil
		}
	}
	return nil, fmt.Errorf("No GPU with UUID" + uuid)
}

func (gpuCollector *GPUCollector) UpdateGPUStatus() error {
	Logger.Info("Updating GPU status")
	_, err := os.Stat(gpu.SocketPath)
	if os.IsNotExist(err) {
		Logger.Error("Can not found ", gpu.SocketPath)
		Logger.Error(err)
		return err
	}
	conn, cleanup, err := connectToServer(gpu.SocketPath)
	if err != nil {
		Logger.Error("Can not connect to ", gpu.SocketPath)
		Logger.Error(err)
		return err
	}

	defer cleanup()
	listPodResp, err := ListPods(conn)
	if err != nil {
		Logger.Error("Can not connect to ", gpu.SocketPath)
		Logger.Error(err)
		return err
	}

	gpuCollector.resetGPUStatus()
	for _, pod := range listPodResp.GetPodResources() {
		for _, container := range pod.GetContainers() {
			for _, dev := range container.GetDevices() {

				if dev.GetResourceName() != gpu.NvidiaResourceName {
					continue
				}

				for _, uuid := range dev.GetDeviceIds() {
					if nvidiaGPU, err := gpuCollector.GetGPUByUUID(uuid); err != nil {
						Logger.Error("No GPU with UUID: ", uuid)
						return err
					} else {
						nvidiaGPU.State = device.GPU_ALLOCATED_STATE
						nvidiaGPU.PodName = pod.Name
						nvidiaGPU.Namespace = pod.Namespace
						Logger.Debug("GPU: ", nvidiaGPU.DeviceFilePath, " allocated to Pod: ", pod.Name, " in Namespace ", pod.Namespace)
					}
				}
			}
		}
	}
	Logger.Info("GPU status update successfully")
	return nil
}

func (gpuCollector *GPUCollector) resetGPUStatus() {
	for _, gpuDev := range gpuCollector.GPUList {
		gpuDev.ResetState()
	}
}

/**
get gpu resources of pod and it slave pod
*/
func (gpuCollector *GPUCollector) GetPodGPUResources(podName string, namespace string) ([]*device.NvidiaGPU, error) {
	err := gpuCollector.UpdateGPUStatus()
	if err != nil {
		Logger.Error("Failed to update gpu status")
		return nil, err
	}
	var gpuResources []*device.NvidiaGPU
	for _, gpuDev := range gpuCollector.GPUList {
		if (gpuDev.PodName == podName && gpuDev.Namespace == namespace) ||
			(strings.Contains(gpuDev.PodName, podName+"-slave-pod-") && gpuDev.Namespace == gpu.GPUPoolNamespace) {
			gpuResources = append(gpuResources, gpuDev)
		}
	}
	return gpuResources, nil
}

func connectToServer(socket string) (*grpc.ClientConn, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), gpu.ConnectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, socket, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, func() {}, fmt.Errorf("failure connecting to %s: %v", socket, err)
	}

	return conn, func() { conn.Close() }, nil
}

func ListPods(conn *grpc.ClientConn) (*podresourcesapi.ListPodResourcesResponse, error) {
	client := podresourcesapi.NewPodResourcesListerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), gpu.ConnectionTimeout)
	defer cancel()

	resp, err := client.List(ctx, &podresourcesapi.ListPodResourcesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failure getting pod resources %v", err)
	}

	return resp, nil
}
