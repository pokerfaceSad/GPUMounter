package device

import (
	"GPUMounter/pkg/util/gpu/collector/nvml"
	. "GPUMounter/pkg/util/log"
	"encoding/json"
	"strconv"
)

type NvidiaGPU struct {
	MinorNumber    int
	DeviceFilePath string
	UUID           string
	State          Devicestate
	PodName        string
	Namespace      string
}
type Devicestate string

const (
	GPU_FREE_STATE      = "GPU_FREE_STATE"
	GPU_ALLOCATED_STATE = "GPU_ALLOCATED_STATE"
)

func New(minorNumber int, uuid string) *NvidiaGPU {
	return &NvidiaGPU{
		MinorNumber:    minorNumber,
		DeviceFilePath: NVIDIA_DEVICE_FILE_PREFIX + strconv.Itoa(minorNumber),
		UUID:           uuid,
		State:          GPU_FREE_STATE,
		PodName:        "",
		Namespace:      "",
	}
}

const (
	DEFAULT_NVIDA_MAJOR_NUMBER     = 195
	DEFAULT_CGROUP_PERMISSION      = "rw"
	DEFAULT_DEVICE_FILE_PERMISSION = "666"
	NVIDIA_DEVICE_FILE_PREFIX      = "/dev/nvidia"
)

func (gpu *NvidiaGPU) String() string {
	out, err := json.Marshal(gpu)
	if err != nil {
		Logger.Error("Failed to parse gpu object to json")
		return "Failed to parse gpu object to json"
	}
	return string(out)
}

func (gpu *NvidiaGPU) ResetState() {
	gpu.PodName = ""
	gpu.Namespace = ""
	gpu.State = GPU_FREE_STATE
}

func (gpu *NvidiaGPU) GetRunningProcess() ([]*nvml.ProcessInfo, error) {
	if err := nvml.Init(); err != nil {
		Logger.Error("nvml error: %+v", err)
		return nil, err
	}
	defer nvml.Shutdown()
	handle, err := nvml.DeviceGetHandleByUUID(gpu.UUID)
	if err != nil {
		Logger.Error(err)
		return nil, err
	}
	uuid, err := handle.DeviceGetUUID()
	if err != nil {
		Logger.Error(err)
		Logger.Info(uuid)
	}
	graphicsProcesses, err := handle.GetGraphicsRunningProcesses(1024)
	if err != nil {
		Logger.Error("Failed to get running graphics processes on GPU: ", gpu.DeviceFilePath)
		Logger.Error(err)
		return nil, err
	}
	computeProcesses, err := handle.GetComputeRunningProcesses(1024)
	if err != nil {
		Logger.Error("Failed to get running compute processes on GPU: ", gpu.DeviceFilePath)
		Logger.Error(err)
		return nil, err
	}
	return append(graphicsProcesses, computeProcesses...), nil
}
