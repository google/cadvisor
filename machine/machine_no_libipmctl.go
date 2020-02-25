// +build !libipmctl

package machine

// GetNVMAvgPowerBudget retrieves configured power budget for NVM devices.
// When libipmct is not available zero is returned.
func GetNVMAvgPowerBudget() (uint, error) {
	return uint(0), nil
}
