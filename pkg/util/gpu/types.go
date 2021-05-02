package gpu

import "time"

const (
	SocketDir  = "/var/lib/kubelet/pod-resources"
	SocketPath = SocketDir + "/kubelet.sock"

	ConnectionTimeout  = 10 * time.Second
	NvidiaResourceName = "nvidia.com/gpu"

	InsufficientGPU     = "InsufficientGPU"
	SuccessfullyCreated = "SuccessfullyCreated"
	FailedCreated       = "FailedCreated"
	SuccessfullyDeleted = "SuccessfullyDeleted"
	FailedDeleted       = "FailedDeleted"

	GPUPoolNamespace = "gpu-pool"
)

type MountType string

const (
	EntireMount MountType = "entire-mount"
	SingleMount MountType = "single-mount"
	NoMount MountType = "no-mount"
	UnknownMount MountType = "unknown-mount"
)