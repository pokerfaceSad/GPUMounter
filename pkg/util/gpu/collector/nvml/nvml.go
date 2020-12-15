// Copyright (c) 2015-2018, NVIDIA CORPORATION. All rights reserved.

package nvml

// #include "nvml.h"
import "C"

type Device struct {
	Handle
}

type ProcessInfo struct {
	Pid           uint
	UsedGPUMemory uint64
}

func (h Handle) DeviceGetMinorNumber() (uint, error) {
	var minor C.uint

	r := C.nvmlDeviceGetMinorNumber(h.dev, &minor)

	return uint(minor), errorString(r)
}

func (h Handle) DeviceGetUUID() (string, error) {
	var uuid [szUUID]C.char

	r := C.nvmlDeviceGetUUID(h.dev, &uuid[0], C.uint(szUUID))

	return C.GoString(&uuid[0]), errorString(r)
}

func (h Handle) GetComputeRunningProcesses(size int) ([]*ProcessInfo, error) {
	var procs = make([]C.nvmlProcessInfo_t, size)
	var count = C.uint(size)

	r := C.nvmlDeviceGetComputeRunningProcesses(h.dev, &count, &procs[0])
	if r != C.NVML_SUCCESS {
		return nil, errorString(r)
	}

	n := int(count)
	info := make([]*ProcessInfo, n)
	for i := 0; i < n; i++ {
		info[i] = &ProcessInfo{
			Pid:           uint(procs[i].pid),
			UsedGPUMemory: uint64(procs[i].usedGpuMemory),
		}
	}

	return info, nil
}

func (h Handle) GetGraphicsRunningProcesses(size int) ([]*ProcessInfo, error) {
	var procs = make([]C.nvmlProcessInfo_t, size)
	var count = C.uint(size)

	r := C.nvmlDeviceGetGraphicsRunningProcesses(h.dev, &count, &procs[0])
	if r != C.NVML_SUCCESS {
		return nil, errorString(r)
	}

	n := int(count)
	info := make([]*ProcessInfo, n)
	for i := 0; i < n; i++ {
		info[i] = &ProcessInfo{
			Pid:           uint(procs[i].pid),
			UsedGPUMemory: uint64(procs[i].usedGpuMemory),
		}
	}

	return info, nil
}

func Init() error {
	return init_()
}

func Shutdown() error {
	return shutdown()
}

func GetDeviceCount() (uint, error) {
	var n C.uint

	r := C.nvmlDeviceGetCount(&n)
	return uint(n), errorString(r)
}

func GetDriverVersion() (string, error) {
	var driver [szDriver]C.char

	r := C.nvmlSystemGetDriverVersion(&driver[0], szDriver)
	return C.GoString(&driver[0]), errorString(r)
}

func GetNVMLVersion() (string, error) {
	var driver [szDriver]C.char

	r := C.nvmlSystemGetNVMLVersion(&driver[0], szDriver)

	return C.GoString(&driver[0]), errorString(r)
}

func DeviceGetHandleByIndex(idx uint) (Handle, error) {
	var dev C.nvmlDevice_t

	r := C.nvmlDeviceGetHandleByIndex(C.uint(idx), &dev)

	return Handle{dev}, errorString(r)
}

func DeviceGetHandleByUUID(uuid string) (Handle, error) {
	var dev C.nvmlDevice_t

	r := C.nvmlDeviceGetHandleByUUID(C.CString(uuid), &dev)

	return Handle{dev}, errorString(r)
}
