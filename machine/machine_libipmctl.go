// +build libipmctl

package machine

// #cgo pkg-config: libipmctl
// #include <nvm_management.h>
import "C"
import (
	"fmt"

	"k8s.io/klog"
)

// GetNVMAvgPowerBudget retrieves configured power budget for NVM devices.
// It uses ipmlibctl to get setting configured in BIOS.
func GetNVMAvgPowerBudget() (uint, error) {
	// Initialize libipmctl library.
	err := C.nvm_init()
	if err != C.NVM_SUCCESS {
		klog.Warningf("libipmctl initialization failed with status %d", err)
		return 0, fmt.Errorf("libipmctl initialization failed with status %d", err)
	}
	defer C.nvm_uninit()

	// Get number of devices on the platform
	// see: https://github.com/intel/ipmctl/blob/v01.00.00.3497/src/os/nvm_api/nvm_management.h#L1478
	var count C.uint
	err = C.nvm_get_number_of_devices(&count)
	if err != C.NVM_SUCCESS {
		klog.Warningf("Unable to get number of NVM devices. Status code: %d", err)
		return uint(0), fmt.Errorf("Unable to get number of NVM devices. Status code: %d", err)
	}

	// Load basic device information for all the devices
	// to obtain UID of the first one.
	var devices = make([]C.struct_device_discovery, count)
	err = C.nvm_get_devices(&devices[0], C.uchar(count))
	if err != C.NVM_SUCCESS {
		klog.Warningf("Unable to get all NVM devices. Status code: %d", err)
		return uint(0), fmt.Errorf("Unable to get all NVM devices. Status code: %d", err)
	}

	// Power budget is same for all the devices
	// so we can rely on any of them.
	var device C.struct_device_details
	err = C.nvm_get_device_details(&devices[0].uid[0], &device)
	if err != C.NVM_SUCCESS {
		uid := C.GoString(&devices[0].uid[0])
		klog.Warningf("Unable to get details of NVM device %q. Status code: %d", uid, err)
		return uint(0), fmt.Errorf("Unable to get details of NVM device %q. Status code: %d", uid, err)
	}

	return uint(device.avg_power_budget), nil
}
