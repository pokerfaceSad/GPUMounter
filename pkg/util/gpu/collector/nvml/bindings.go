/*
 * Copyright (c) 2020, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nvml

/*
#cgo linux LDFLAGS: -ldl -Wl,--unresolved-symbols=ignore-in-object-files
#cgo darwin LDFLAGS: -ldl -Wl,-undefined,dynamic_lookup
#cgo windows LDFLAGS: -LC:/Program\ Files/NVIDIA\ Corporation/NVSMI -lnvml
#include "nvml.h"

#undef nvmlEventSetWait
nvmlReturn_t DECLDIR nvmlEventSetWait(nvmlEventSet_t set, nvmlEventData_t * data, unsigned int timeoutms);
nvmlReturn_t DECLDIR nvmlEventSetWait_v2(nvmlEventSet_t set, nvmlEventData_t * data, unsigned int timeoutms);
*/
import "C"

import (
	"errors"
	"fmt"
)

const (
	szDriver = C.NVML_SYSTEM_DRIVER_VERSION_BUFFER_SIZE
	szUUID   = C.NVML_DEVICE_UUID_V2_BUFFER_SIZE
)

type Handle struct{ dev C.nvmlDevice_t }

func errorString(ret C.nvmlReturn_t) error {
	if ret == C.NVML_SUCCESS {
		return nil
	}
	err := C.GoString(C.nvmlErrorString(ret))
	return fmt.Errorf("nvml: %v", err)
}

func init_() error {
	r := dl.nvmlInit()
	if r == C.NVML_ERROR_LIBRARY_NOT_FOUND {
		return errors.New("could not load NVML library")
	}
	return errorString(r)
}

func shutdown() error {
	return errorString(dl.nvmlShutdown())
}
