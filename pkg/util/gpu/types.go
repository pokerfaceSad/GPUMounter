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
)
