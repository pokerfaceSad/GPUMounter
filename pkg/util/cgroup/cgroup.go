package cgroup

import (
	"GPUMounter/pkg/device"
	. "GPUMounter/pkg/util/log"
	"bufio"
	"fmt"
	cgroupsystemd "github.com/opencontainers/runc/libcontainer/cgroups/systemd"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

// CgroupName is the abstract name of a cgroup prior to any driver specific conversion.
// It is specified as a list of strings from its individual components, such as:
// {"kubepods", "burstable", "pod1234-abcd-5678-efgh"}
type CgroupName []string

const (
	// systemdSuffix is the cgroup name suffix for systemd
	systemdSuffix string = ".slice"
)

// NewCgroupName composes a new cgroup name.
// Use RootCgroupName as base to start at the root.
// This function does some basic check for invalid characters at the name.
func NewCgroupName(base CgroupName, components ...string) CgroupName {
	for _, component := range components {
		// Forbit using "_" in internal names. When remapping internal
		// names to systemd cgroup driver, we want to remap "-" => "_",
		// so we forbid "_" so that we can always reverse the mapping.
		if strings.Contains(component, "/") || strings.Contains(component, "_") {
			panic(fmt.Errorf("invalid character in component [%q] of CgroupName", component))
		}
	}
	// copy data from the base cgroup to eliminate cases where CgroupNames share underlying slices.  See #68416
	baseCopy := make([]string, len(base))
	copy(baseCopy, base)
	return CgroupName(append(baseCopy, components...))
}

// cgroupName.ToSystemd converts the internal cgroup name to a systemd name.
// For example, the name {"kubepods", "burstable", "pod1234-abcd-5678-efgh"} becomes
// "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod1234_abcd_5678_efgh.slice"
// This function always expands the systemd name into the cgroupfs form. If only
// the last part is needed, use path.Base(...) on it to discard the rest.
func (cgroupName CgroupName) ToSystemd() string {
	if len(cgroupName) == 0 || (len(cgroupName) == 1 && cgroupName[0] == "") {
		return "/"
	}
	newparts := []string{}
	for _, part := range cgroupName {
		part = escapeSystemdCgroupName(part)
		newparts = append(newparts, part)
	}

	result, err := cgroupsystemd.ExpandSlice(strings.Join(newparts, "-") + systemdSuffix)
	if err != nil {
		// Should never happen...
		panic(fmt.Errorf("error converting cgroup name [%v] to systemd format: %v", cgroupName, err))
	}
	return result
}

func escapeSystemdCgroupName(part string) string {
	return strings.Replace(part, "-", "_", -1)
}

func (cgroupName CgroupName) ToCgroupfs() string {
	return "/" + path.Join(cgroupName...)
}

func GetCgroupName(cgroupDriver string, pod *corev1.Pod, containerID string) (string, error) {
	containerRoot := NewCgroupName([]string{}, "kubepods")
	PodCgroupNamePrefix := "pod"
	podQos := GetPodQOS(pod)

	var parentContainer CgroupName
	switch podQos {
	case corev1.PodQOSGuaranteed:
		parentContainer = NewCgroupName(containerRoot)
	case corev1.PodQOSBurstable:
		parentContainer = NewCgroupName(containerRoot, strings.ToLower(string(corev1.PodQOSBurstable)))
	case corev1.PodQOSBestEffort:
		parentContainer = NewCgroupName(containerRoot, strings.ToLower(string(corev1.PodQOSBestEffort)))
	}

	podContainer := PodCgroupNamePrefix + string(pod.UID)
	cgroupName := NewCgroupName(parentContainer, podContainer)

	switch cgroupDriver {
	case "systemd":
		return fmt.Sprintf("%s/%s-%s.scope", cgroupName.ToSystemd(), "docker", containerID), nil
	case "cgroupfs":
		return fmt.Sprintf("%s/%s", cgroupName.ToCgroupfs(), containerID), nil
	default:
	}

	return "", fmt.Errorf("unsupported cgroup driver")
}

func GetDeviceGroupPath(cgroupPath string) string {
	deviceCgroupPath := "/sys/fs/cgroup/devices" + cgroupPath
	return deviceCgroupPath
}

func GetCgroupPIDs(cgroupPath string) ([]string, error) {
	deviceCgroupPath := GetDeviceGroupPath(cgroupPath)
	procsFileName := "cgroup.procs"
	fil, err := os.Open(deviceCgroupPath + "/" + procsFileName)
	if err != nil {
		Logger.Error("Open " + deviceCgroupPath + "/" + procsFileName + " failed")
		return nil, err
	}
	defer fil.Close()
	var pids []string
	br := bufio.NewReader(fil)
	for {
		pid, err := br.ReadString('\n')
		if err != nil {
			break
		}
		pid = strings.Replace(pid, "\n", "", -1)
		pids = append(pids, pid)

	}
	return pids, nil
}

func AddGPUDevicePermission(cgroupPath string, gpu *device.NvidiaGPU) error {
	deviceCgroupPath := GetDeviceGroupPath(cgroupPath)
	cmd := "echo 'c " + strconv.Itoa(device.DEFAULT_NVIDA_MAJOR_NUMBER) + ":" + strconv.Itoa(gpu.MinorNumber) + " " + device.DEFAULT_CGROUP_PERMISSION + "' > " + deviceCgroupPath + "/devices.allow"
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		Logger.Error("Exec \"" + cmd + "\" failed")
		Logger.Error("Output: " + string(out))
		Logger.Error(err)
		return err
	} else {
		return nil
	}
}

func RemoveGPUDevicePermission(cgroupPath string, gpu *device.NvidiaGPU) error {
	deviceCgroupPath := GetDeviceGroupPath(cgroupPath)
	cmd := "echo 'c " + strconv.Itoa(device.DEFAULT_NVIDA_MAJOR_NUMBER) + ":" + strconv.Itoa(gpu.MinorNumber) + " " + device.DEFAULT_CGROUP_PERMISSION + "' > " + deviceCgroupPath + "/devices.deny"
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		Logger.Error("Exec \"" + cmd + "\" failed")
		Logger.Error("Output: " + string(out))
		Logger.Error(err)
		return err
	} else {
		return nil
	}
}

var supportedQoSComputeResources = sets.NewString(string(corev1.ResourceCPU), string(corev1.ResourceMemory))

func isSupportedQoSComputeResource(name corev1.ResourceName) bool {
	return supportedQoSComputeResources.Has(string(name))
}

func GetPodQOS(pod *corev1.Pod) corev1.PodQOSClass {
	requests := corev1.ResourceList{}
	limits := corev1.ResourceList{}
	zeroQuantity := resource.MustParse("0")
	isGuaranteed := true
	for _, container := range pod.Spec.Containers {
		// process requests
		for name, quantity := range container.Resources.Requests {
			if !isSupportedQoSComputeResource(name) {
				continue
			}
			if quantity.Cmp(zeroQuantity) == 1 {
				delta := quantity.DeepCopy()
				if _, exists := requests[name]; !exists {
					requests[name] = delta
				} else {
					delta.Add(requests[name])
					requests[name] = delta
				}
			}
		}
		// process limits
		qosLimitsFound := sets.NewString()
		for name, quantity := range container.Resources.Limits {
			if !isSupportedQoSComputeResource(name) {
				continue
			}
			if quantity.Cmp(zeroQuantity) == 1 {
				qosLimitsFound.Insert(string(name))
				delta := quantity.DeepCopy()
				if _, exists := limits[name]; !exists {
					limits[name] = delta
				} else {
					delta.Add(limits[name])
					limits[name] = delta
				}
			}
		}

		if !qosLimitsFound.HasAll(string(corev1.ResourceMemory), string(corev1.ResourceCPU)) {
			isGuaranteed = false
		}
	}
	if len(requests) == 0 && len(limits) == 0 {
		return corev1.PodQOSBestEffort
	}
	// Check is requests match limits for all resources.
	if isGuaranteed {
		for name, req := range requests {
			if lim, exists := limits[name]; !exists || lim.Cmp(req) != 0 {
				isGuaranteed = false
				break
			}
		}
	}
	if isGuaranteed &&
		len(requests) == len(limits) {
		return corev1.PodQOSGuaranteed
	}
	return corev1.PodQOSBurstable
}
