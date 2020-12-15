package main

import (
	gpu_mount_api "GPUMounter/pkg/api/gpu-mount"
	gpu_mount "GPUMounter/pkg/server/gpu-mount"
	. "GPUMounter/pkg/util/log"
	"google.golang.org/grpc"
	"net"
)

func main() {
	InitLogger("/var/log/GPUMounter/", "GPUMounter-worker.log")
	defer Logger.Sync()

	Logger.Info("Service Starting...")
	gpuMounter, err := gpu_mount.NewGPUMounter()
	if err != nil {
		Logger.Error("Failed to init gpu mounter")
		Logger.Error(err)
		return
	}
	Logger.Info("Successfully created gpu mounter")

	lis, err := net.Listen("tcp", ":1200")
	if err != nil {
		Logger.Error("Listen Port Failed")
		Logger.Error(err)
	}

	s := grpc.NewServer()
	gpu_mount_api.RegisterAddGPUServiceServer(s, gpuMounter)
	gpu_mount_api.RegisterRemoveGPUServiceServer(s, gpuMounter)
	err = s.Serve(lis)
	if err != nil {
		Logger.Error("service start failed")
		Logger.Error(err)
		return
	}
}
