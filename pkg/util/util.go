package util

import (
	gpu_mount "GPUMounter/pkg/api/gpu-mount"
	"GPUMounter/pkg/device"
	"GPUMounter/pkg/util/cgroup"
	. "GPUMounter/pkg/util/log"
	"GPUMounter/pkg/util/namespace"
	"errors"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

func MountGPU(pod *corev1.Pod, gpu *device.NvidiaGPU) error {

	Logger.Info("Start mount GPU: " + gpu.String() + " to Pod: " + pod.Name)

	// change devices control group
	containerID := pod.Status.ContainerStatuses[0].ContainerID
	containerID = strings.Replace(containerID, "docker://", "", 1)
	Logger.Info("Pod :" + pod.Name + " container ID: " + containerID)
	cgroupPath, err := cgroup.GetCgroupName("cgroupfs", pod, containerID)
	if err != nil {
		Logger.Error("Get cgroup path for Pod: " + pod.Name + " failed")
		return err
	}
	Logger.Info("Successfully get cgroup path: " + cgroupPath + " for Pod: " + pod.Name)

	if err := cgroup.AddGPUDevicePermission(cgroupPath, gpu); err != nil {
		Logger.Error("Add GPU " + gpu.String() + "failed")
		return err
	}
	Logger.Info("Successfully add GPU: " + gpu.String() + " permisssion for Pod: " + pod.Name)

	// get target PID of this group
	pids, err := cgroup.GetCgroupPIDs(cgroupPath)
	if err != nil {
		Logger.Error("Get PID of Pod: " + pod.Name + " Container: " + containerID + " failed")
		Logger.Error(err)
		return err
	}
	PID, err := strconv.Atoi(pids[0])
	if err != nil {
		Logger.Error("Invalid PID: ", pids[0])
		Logger.Error(err)
		return err
	}

	Logger.Info("Successfully get PID: " + strconv.Itoa(PID) + " of Pod: " + pod.Name + " Container: " + containerID)

	// enter container namespace to mknod
	cfg := &namespace.Config{
		Mount:  true, // Execute into mount namespace
		Target: PID,  // Enter into Target namespace
	}
	if err := namespace.AddGPUDeviceFile(cfg, gpu); err != nil {
		Logger.Error("Failed to create device file in Target PID Namespace: ", PID, " Pod: ", pod.Name, " Namespace: ", pod.Namespace)
		return err
	}
	Logger.Info("Successfully create device file in Target PID Namespace: ", PID, " Pod: ", pod.Name, " Namespace: ", pod.Namespace)
	return nil

}

func UnmountGPU(pod *corev1.Pod, gpu *device.NvidiaGPU, forceRemove bool) error {
	Logger.Info("Start unmount GPU: " + gpu.String() + " from Pod: " + pod.Name)

	// get devices control group
	containerID := pod.Status.ContainerStatuses[0].ContainerID
	containerID = strings.Replace(containerID, "docker://", "", 1)
	Logger.Info("Pod :" + pod.Name + " container ID: " + containerID)
	cgroupPath, err := cgroup.GetCgroupName("cgroupfs", pod, containerID)
	if err != nil {
		Logger.Error("Get cgroup path for Pod: " + pod.Name + " failed")
		return err
	}
	Logger.Info("Successfully get cgroup path: " + cgroupPath + " for Pod: " + pod.Name)

	// get running processes
	pids, err := cgroup.GetCgroupPIDs(cgroupPath)
	if err != nil {
		Logger.Error("Failed to get running processes in Pod: ", pod.Name, " Namespace: ", pod.Namespace)
		Logger.Error(err)
		return err
	}

	podGPUProcesses, err := GetPodGPUProcesses(pod, gpu)
	if err != nil {
		Logger.Error("Failed to get GPU: ", gpu.DeviceFilePath+" status in Pod: ", pod.Name, " in Namespace: ", pod.Namespace)
		Logger.Error(err)
		return err
	}
	if podGPUProcesses != nil && !forceRemove {
		Logger.Info("GPU: ", gpu.DeviceFilePath, " status in Pod: ", pod.Name, " in Namespace: ", pod.Namespace, " is busy")
		return errors.New(string(gpu_mount.RemoveGPUResponse_GPUBusy))
	}

	// remove permission
	if err := cgroup.RemoveGPUDevicePermission(cgroupPath, gpu); err != nil {
		Logger.Error("Remove GPU " + gpu.String() + "failed")
		return err
	}

	// delete device files
	PID, err := strconv.Atoi(pids[0])
	if err != nil {
		Logger.Error("Invalid PID: ", pids[0])
		Logger.Error(err)
		return err
	}
	Logger.Info("Successfully get PID: " + strconv.Itoa(PID) + " of Pod: " + pod.Name + " Container: " + containerID)

	// enter container namespace
	cfg := &namespace.Config{
		Mount:  true, // Execute into mount namespace
		Target: PID,  // Enter into Target namespace
	}
	if err := namespace.RemoveGPUDeviceFile(cfg, gpu); err != nil {
		Logger.Error("Failed to remove device file in Target PID Namespace: ", PID, " Pod: ", pod.Name, " Namespace: ", pod.Namespace)
		return err
	}

	// kill all running procs
	if podGPUProcesses != nil {
		Logger.Info("Killing running gpu Processes", strings.Join(podGPUProcesses, ", "), " on Pod: ", pod.Name, " Namespace: ", pod.Namespace)
		if err := namespace.KillRunningGPUProcesses(cfg, podGPUProcesses); err != nil {
			Logger.Error("Failed to kill gpu processes in Target PID Namespace: ", PID, " Pod: ", pod.Name, " Namespace: ", pod.Namespace)
			return err
		}
	} else {
		Logger.Info("No running gpu process on Pod: ", pod.Name, " Namespace: ", pod.Namespace)
	}
	return nil
}

/**
get all gpu proc pid in pod, return nil if no gpu proc in pod
*/
func GetPodGPUProcesses(pod *corev1.Pod, gpu *device.NvidiaGPU) ([]string, error) {
	// get devices control group
	containerID := pod.Status.ContainerStatuses[0].ContainerID
	containerID = strings.Replace(containerID, "docker://", "", 1)
	Logger.Info("Pod: " + pod.Name + " container ID: " + containerID)
	cgroupPath, err := cgroup.GetCgroupName("cgroupfs", pod, containerID)
	if err != nil {
		Logger.Error("Get cgroup path for Pod: " + pod.Name + " failed")
		return nil, err
	}
	Logger.Debug("Successfully get cgroup path: " + cgroupPath + " for Pod: " + pod.Name)

	// get running processes
	podProcess, err := cgroup.GetCgroupPIDs(cgroupPath)
	if err != nil {
		Logger.Error("Failed to get running processes in Pod: ", pod.Name, " Namespace: ", pod.Namespace)
		Logger.Error(err)
		return nil, err
	}

	gpuProcess, err := gpu.GetRunningProcess()
	if err != nil {
		Logger.Error("Failed to get process info on GPU: ", gpu.DeviceFilePath)
		Logger.Error(err)
		return nil, err
	}

	var podGPUProcess []string
	for _, processInfo := range gpuProcess {
		if ContainString(podProcess, strconv.Itoa(int(processInfo.Pid))) {
			podGPUProcess = append(podGPUProcess, strconv.Itoa(int(processInfo.Pid)))
		}
	}
	if len(podGPUProcess) != 0 {
		Logger.Debug("{Namespace: ", pod.Namespace, " Pod: ", pod.Name, "}proc PID: ", strings.Join(podGPUProcess, ", "), " running on GPU: ", gpu.UUID)
		return podGPUProcess, nil
	}
	Logger.Debug("{Namespace: ", pod.Namespace, " Pod: ", pod.Name, "} has no proc running on GPU: ", gpu.UUID)
	return nil, nil
}

func ContainString(stringList []string, aimString string) bool {
	for _, str := range stringList {
		if str == aimString {
			return true
		}
	}
	return false
}
