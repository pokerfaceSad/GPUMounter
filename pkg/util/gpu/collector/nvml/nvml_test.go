package nvml

import (
	"fmt"
	"testing"
)

func check(err error, t *testing.T) {
	if err != nil {
		t.Errorf("%v\n", err)
	}
}

func TestDeviceCount(t *testing.T) {
	Init()
	defer Shutdown()

	count, err := GetDeviceCount()
	check(err, t)

	fmt.Println(count)
}

func TestGetDriverVersion(t *testing.T) {
	Init()
	defer Shutdown()

	ver, err := GetDriverVersion()
	check(err, t)

	fmt.Println(ver)
}

func TestGetNVMLVersion(t *testing.T) {
	Init()
	defer Shutdown()

	ver, err := GetNVMLVersion()
	check(err, t)

	fmt.Println(ver)
}

func TestDeviceGetHandleByIndex(t *testing.T) {
	Init()
	defer Shutdown()

	handle, err := DeviceGetHandleByIndex(0)
	check(err, t)

	fmt.Println(handle)
}

func TestHandle_DeviceGetMinorNumber(t *testing.T) {
	Init()
	defer Shutdown()

	handle, err := DeviceGetHandleByIndex(2)
	check(err, t)

	minorNumber, err := handle.DeviceGetMinorNumber()
	check(err, t)

	fmt.Println(minorNumber)
}

func TestHandle_DeviceGetUUID(t *testing.T) {
	Init()
	defer Shutdown()

	handle, err := DeviceGetHandleByIndex(2)
	check(err, t)

	uuid, err := handle.DeviceGetUUID()
	check(err, t)

	fmt.Println(uuid)
}
