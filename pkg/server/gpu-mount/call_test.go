package gpu_mount

import (
	gpu_mount "GPUMounter/pkg/api/gpu-mount"
	"context"
	"google.golang.org/grpc"
	"log"
	"testing"
)

func TestGpuMountImpl_AddGPU(t *testing.T) {

	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost:1200", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := gpu_mount.NewAddGPUServiceClient(conn)

	// Contact the server and print out its response.
	_, err = c.AddGPU(context.TODO(), &gpu_mount.AddGPURequest{
		PodName:              "gpu-pod2",
		Namespace:            "default",
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	})

	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

}
