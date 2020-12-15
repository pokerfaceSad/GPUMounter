package collector

import (
	. "GPUMounter/pkg/util/log"
	"testing"
)

func TestGetGPUInfo(t *testing.T) {
	InitLogger(".", "log")
	defer Logger.Sync()

	gpuCollector, err := NewGPUCollector()
	if err != nil {
		Logger.Error("Failed to get gpus info")
		panic(err)
	}
	for _, gpu := range gpuCollector.GPUList {
		Logger.Info(gpu)
	}

}

func TestGPUCollector_UpdateGPUStatus(t *testing.T) {
	InitLogger(".", "log")
	defer Logger.Sync()

	gpuCollector, err := NewGPUCollector()
	if err != nil {
		Logger.Error(err)
		panic(err)
	}
	err = gpuCollector.UpdateGPUStatus()
	if err != nil {
		Logger.Error(err)
		panic(err)
	}
	for _, gpuDev := range gpuCollector.GPUList {
		Logger.Info(gpuDev)
	}
}

func TestGPUCollector_GetPodGPUResources(t *testing.T) {
	InitLogger(".", "log")
	defer Logger.Sync()

	gpuCollector, err := NewGPUCollector()
	if err != nil {
		Logger.Error(err)
		panic(err)
	}
	gpuResources, err := gpuCollector.GetPodGPUResources("gpu-pod2", "default")
	if err != nil {
		Logger.Error(err)
		panic(err)
	}
	for _, gpuResource := range gpuResources {
		procs, err := gpuResource.GetRunningProcess()
		if err != nil {
			Logger.Error(err)
			panic(err)
		}
		for _, proc := range procs {
			Logger.Info(proc.Pid)
		}

	}
}
