package util

import (
	gpu_mount "GPUMounter/pkg/api/gpu-mount"
	"GPUMounter/pkg/device"
	"GPUMounter/pkg/util/cgroup"
	"GPUMounter/pkg/util/gpu/collector/nvml"
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
		return nil
	}

	processInfos, err := gpu.GetRunningProcess()
	if err != nil {
		Logger.Error("Failed to get process info on GPU: ", gpu.DeviceFilePath)
		Logger.Error(err)
		return err
	}

	podGPUProcesses := getPodGPUProcess(pids, processInfos)
	if len(podGPUProcesses) != 0 {
		if !forceRemove {
			return errors.New(string(gpu_mount.RemoveGPUResponse_GPUBusy))
		}
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
	if len(podGPUProcesses) != 0 {
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

func getPodGPUProcess(podPIDS []string, processInfos []*nvml.ProcessInfo) []string {
	var gpuProcess []string
	for _, processInfo := range processInfos {
		if ContainString(podPIDS, strconv.Itoa(int(processInfo.Pid))) {
			gpuProcess = append(gpuProcess, strconv.Itoa(int(processInfo.Pid)))
		}
	}
	return gpuProcess
}

func ContainString(stringList []string, aimString string) bool {
	for _, str := range stringList {
		if str == aimString {
			return true
		}
	}
	return false
}
