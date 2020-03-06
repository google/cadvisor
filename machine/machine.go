// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The machine package contains functions that extract machine-level specs.
package machine

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	// s390/s390x changes
	"runtime"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/utils/sysinfo"

	"k8s.io/klog"

	"golang.org/x/sys/unix"
)

var (
	cpuRegExp     = regexp.MustCompile(`^processor\s*:\s*([0-9]+)$`)
	coreRegExp    = regexp.MustCompile(`(?m)^core id\s*:\s*([0-9]+)$`)
	nodeRegExp    = regexp.MustCompile(`(?m)^physical id\s*:\s*([0-9]+)$`)
	nodeBusRegExp = regexp.MustCompile(`^node([0-9]+)$`)
	// Power systems have a different format so cater for both
	cpuClockSpeedMHz     = regexp.MustCompile(`(?:cpu MHz|clock)\s*:\s*([0-9]+\.[0-9]+)(?:MHz)?`)
	memoryCapacityRegexp = regexp.MustCompile(`MemTotal:\s*([0-9]+) kB`)
	swapCapacityRegexp   = regexp.MustCompile(`SwapTotal:\s*([0-9]+) kB`)

	cpuBusPath         = "/sys/bus/cpu/devices/"
	isMemoryController = regexp.MustCompile("mc[0-9]+")
	isDimm             = regexp.MustCompile("dimm[0-9]+")
)

const maxFreqFile = "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq"
const nodePath = "/sys/devices/system/node"
const sysFsCPUCoreID = "core_id"
const sysFsCPUPhysicalPackageID = "physical_package_id"
const sysFsCPUTopology = "topology"
const memTypeFileName = "dimm_mem_type"
const sizeFileName = "size"

// GetPhysicalCores returns number of CPU cores reading /proc/cpuinfo file or if needed information from sysfs cpu path
func GetPhysicalCores(procInfo []byte) int {
	numCores := getUniqueMatchesCount(string(procInfo), coreRegExp)
	if numCores == 0 {
		// read number of cores from /sys/bus/cpu/devices/cpu*/topology/core_id to deal with processors
		// for which 'core id' is not available in /proc/cpuinfo
		numCores = getUniqueCPUPropertyCount(cpuBusPath, sysFsCPUCoreID)
	}
	if numCores == 0 {
		klog.Errorf("Cannot read number of physical cores correctly, number of cores set to %d", numCores)
	}
	return numCores
}

// GetSockets returns number of CPU sockets reading /proc/cpuinfo file or if needed information from sysfs cpu path
func GetSockets(procInfo []byte) int {
	numSocket := getUniqueMatchesCount(string(procInfo), nodeRegExp)
	if numSocket == 0 {
		// read number of sockets from /sys/bus/cpu/devices/cpu*/topology/physical_package_id to deal with processors
		// for which 'physical id' is not available in /proc/cpuinfo
		numSocket = getUniqueCPUPropertyCount(cpuBusPath, sysFsCPUPhysicalPackageID)
	}
	if numSocket == 0 {
		klog.Errorf("Cannot read number of sockets correctly, number of sockets set to %d", numSocket)
	}
	return numSocket
}

// GetClockSpeed returns the CPU clock speed, given a []byte formatted as the /proc/cpuinfo file.
func GetClockSpeed(procInfo []byte) (uint64, error) {
	// s390/s390x, mips64, riscv64, aarch64 and arm32 changes
	if isMips64() || isSystemZ() || isAArch64() || isArm32() || isRiscv64() {
		return 0, nil
	}

	// First look through sys to find a max supported cpu frequency.
	if utils.FileExists(maxFreqFile) {
		val, err := ioutil.ReadFile(maxFreqFile)
		if err != nil {
			return 0, err
		}
		var maxFreq uint64
		n, err := fmt.Sscanf(string(val), "%d", &maxFreq)
		if err != nil || n != 1 {
			return 0, fmt.Errorf("could not parse frequency %q", val)
		}
		return maxFreq, nil
	}
	// Fall back to /proc/cpuinfo
	matches := cpuClockSpeedMHz.FindSubmatch(procInfo)
	if len(matches) != 2 {
		return 0, fmt.Errorf("could not detect clock speed from output: %q", string(procInfo))
	}

	speed, err := strconv.ParseFloat(string(matches[1]), 64)
	if err != nil {
		return 0, err
	}
	// Convert to kHz
	return uint64(speed * 1000), nil
}

// GetMachineMemoryCapacity returns the machine's total memory from /proc/meminfo.
// Returns the total memory capacity as an uint64 (number of bytes).
func GetMachineMemoryCapacity() (uint64, error) {
	out, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	memoryCapacity, err := parseCapacity(out, memoryCapacityRegexp)
	if err != nil {
		return 0, err
	}
	return memoryCapacity, err
}

// GetMachineMemoryByType returns information about memory capcity and number of DIMMs.
// Information is retrieved from sysfs edac per-DIMM API (/sys/devices/system/edac/mc/)
// introduced in kernel 3.6. Documentation can be found at
// https://www.kernel.org/doc/Documentation/admin-guide/ras.rst.
// Full list of memory types can be found in edac_mc.c
// (https://github.com/torvalds/linux/blob/v5.5/drivers/edac/edac_mc.c#L198)
func GetMachineMemoryByType(edacPath string) (map[string]*info.MemoryInfo, error) {
	memory := map[string]*info.MemoryInfo{}
	names, err := ioutil.ReadDir(edacPath)
	// On some architectures (such as ARM) memory controller device may not exist.
	// If this is the case then we ignore error and return empty slice.
	_, ok := err.(*os.PathError)
	if err != nil && ok {
		return memory, nil
	} else if err != nil {
		return memory, err
	}
	for _, controllerDir := range names {
		controller := controllerDir.Name()
		if !isMemoryController.MatchString(controller) {
			continue
		}
		dimms, err := ioutil.ReadDir(path.Join(edacPath, controllerDir.Name()))
		if err != nil {
			return map[string]*info.MemoryInfo{}, err
		}
		for _, dimmDir := range dimms {
			dimm := dimmDir.Name()
			if !isDimm.MatchString(dimm) {
				continue
			}
			memType, err := ioutil.ReadFile(path.Join(edacPath, controller, dimm, memTypeFileName))
			readableMemType := strings.TrimSpace(string(memType))
			if err != nil {
				return map[string]*info.MemoryInfo{}, err
			}
			if _, exists := memory[readableMemType]; !exists {
				memory[readableMemType] = &info.MemoryInfo{}
			}
			size, err := ioutil.ReadFile(path.Join(edacPath, controller, dimm, sizeFileName))
			if err != nil {
				return map[string]*info.MemoryInfo{}, err
			}
			capacity, err := strconv.Atoi(strings.TrimSpace(string(size)))
			if err != nil {
				return map[string]*info.MemoryInfo{}, err
			}
			memory[readableMemType].Capacity += uint64(mbToBytes(capacity))
			memory[readableMemType].DimmCount++
		}
	}

	return memory, nil
}

func mbToBytes(megabytes int) int {
	return megabytes * 1024 * 1024
}

// GetMachineSwapCapacity returns the machine's total swap from /proc/meminfo.
// Returns the total swap capacity as an uint64 (number of bytes).
func GetMachineSwapCapacity() (uint64, error) {
	out, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	swapCapacity, err := parseCapacity(out, swapCapacityRegexp)
	if err != nil {
		return 0, err
	}
	return swapCapacity, err
}

// parseCapacity matches a Regexp in a []byte, returning the resulting value in bytes.
// Assumes that the value matched by the Regexp is in KB.
func parseCapacity(b []byte, r *regexp.Regexp) (uint64, error) {
	matches := r.FindSubmatch(b)
	if len(matches) != 2 {
		return 0, fmt.Errorf("failed to match regexp in output: %q", string(b))
	}
	m, err := strconv.ParseUint(string(matches[1]), 10, 64)
	if err != nil {
		return 0, err
	}

	// Convert to bytes.
	return m * 1024, err
}

// Looks for sysfs cpu path containing core_id
// Such as: sys/bus/cpu/devices/cpu0/topology/core_id
func getCoreIdFromCpuBus(cpuBusPath string, threadId int) (int, error) {
	path := filepath.Join(cpuBusPath, fmt.Sprintf("cpu%d/topology", threadId))
	file := filepath.Join(path, sysFsCPUCoreID)

	num, err := ioutil.ReadFile(file)
	if err != nil {
		return threadId, err
	}

	coreId, err := strconv.ParseInt(string(bytes.TrimSpace(num)), 10, 32)
	if err != nil {
		return threadId, err
	}

	if coreId < 0 {
		// report threadId if found coreId < 0
		coreId = int64(threadId)
	}

	return int(coreId), nil
}

// Looks for sysfs cpu path containing given CPU property, e.g. core_id or physical_package_id
// and returns number of unique values of given property, exemplary usage: getting number of CPU physical cores
func getUniqueCPUPropertyCount(cpuBusPath string, propertyName string) int {
	pathPattern := cpuBusPath + "cpu*[0-9]"
	sysCPUPaths, err := filepath.Glob(pathPattern)
	if err != nil {
		klog.Errorf("Cannot find files matching pattern (pathPattern: %s),  number of unique %s set to 0", pathPattern, propertyName)
		return 0
	}
	uniques := make(map[string]bool)
	for _, sysCPUPath := range sysCPUPaths {
		propertyPath := filepath.Join(sysCPUPath, sysFsCPUTopology, propertyName)
		propertyVal, err := ioutil.ReadFile(propertyPath)
		if err != nil {
			klog.Errorf("Cannot open %s, number of unique %s  set to 0", propertyPath, propertyName)
			return 0
		}
		uniques[string(propertyVal)] = true
	}
	return len(uniques)
}

// Looks for sysfs cpu path containing node id
// Such as: /sys/bus/cpu/devices/cpu0/node%d
func getNodeIdFromCpuBus(cpuBusPath string, threadId int) (int, error) {
	path := filepath.Join(cpuBusPath, fmt.Sprintf("cpu%d", threadId))

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return 0, err
	}

	nodeId := 0
	for _, file := range files {
		filename := file.Name()

		ok, val, _ := extractValue(filename, nodeBusRegExp)
		if ok {
			if val < 0 {
				continue
			}
			nodeId = val
			break
		}
	}

	return nodeId, nil
}

// GetHugePagesInfo returns information about pre-allocated huge pages
// hugepagesDirectory should be top directory of hugepages
// Such as: /sys/kernel/mm/hugepages/
func GetHugePagesInfo(hugepagesDirectory string) ([]info.HugePagesInfo, error) {
	var hugePagesInfo []info.HugePagesInfo
	files, err := ioutil.ReadDir(hugepagesDirectory)
	if err != nil {
		// treat as non-fatal since kernels and machine can be
		// configured to disable hugepage support
		return hugePagesInfo, nil
	}
	for _, st := range files {
		nameArray := strings.Split(st.Name(), "-")
		pageSizeArray := strings.Split(nameArray[1], "kB")
		pageSize, err := strconv.ParseUint(string(pageSizeArray[0]), 10, 64)
		if err != nil {
			return hugePagesInfo, err
		}

		numFile := hugepagesDirectory + st.Name() + "/nr_hugepages"
		val, err := ioutil.ReadFile(numFile)
		if err != nil {
			return hugePagesInfo, err
		}
		var numPages uint64
		// we use sscanf as the file as a new-line that trips up ParseUint
		// it returns the number of tokens successfully parsed, so if
		// n != 1, it means we were unable to parse a number from the file
		n, err := fmt.Sscanf(string(val), "%d", &numPages)
		if err != nil || n != 1 {
			return hugePagesInfo, fmt.Errorf("could not parse file %v contents %q", numFile, string(val))
		}

		hugePagesInfo = append(hugePagesInfo, info.HugePagesInfo{
			NumPages: numPages,
			PageSize: pageSize,
		})
	}
	return hugePagesInfo, nil
}

func GetTopology(sysFs sysfs.SysFs, cpuinfo string) ([]info.Node, int, error) {
	nodes := []info.Node{}

	// s390/s390x changes
	if true == isSystemZ() {
		return nodes, getNumCores(), nil
	}

	numCores := 0
	lastThread := -1
	lastCore := -1
	lastNode := -1
	for _, line := range strings.Split(cpuinfo, "\n") {
		if line == "" {
			continue
		}
		ok, val, err := extractValue(line, cpuRegExp)
		if err != nil {
			return nil, -1, fmt.Errorf("could not parse cpu info from %q: %v", line, err)
		}
		if ok {
			thread := val
			numCores++
			if lastThread != -1 {
				// New cpu section. Save last one.
				nodeIdx, err := addNode(&nodes, lastNode)
				if err != nil {
					return nil, -1, fmt.Errorf("failed to add node %d: %v", lastNode, err)
				}
				nodes[nodeIdx].AddThread(lastThread, lastCore)
				lastCore = -1
				lastNode = -1
			}
			lastThread = thread

			/* On Arm platform, no 'core id' and 'physical id' in '/proc/cpuinfo'. */
			/* So we search sysfs cpu path directly. */
			/* This method can also be used on other platforms, such as x86, ppc64le... */
			/* /sys/bus/cpu/devices/cpu%d contains the information of 'core_id' & 'node_id'. */
			/* Such as: /sys/bus/cpu/devices/cpu0/topology/core_id */
			/* Such as:  /sys/bus/cpu/devices/cpu0/node0 */
			if isAArch64() {
				val, err = getCoreIdFromCpuBus(cpuBusPath, lastThread)
				if err != nil {
					// Report thread id if no NUMA
					val = lastThread
				}
				lastCore = val

				val, err = getNodeIdFromCpuBus(cpuBusPath, lastThread)
				if err != nil {
					// Report node 0 if no NUMA
					val = 0
				}
				lastNode = val
			}
			continue
		}

		if isAArch64() {
			/* On Arm platform, no 'core id' and 'physical id' in '/proc/cpuinfo'. */
			continue
		}

		ok, val, err = extractValue(line, coreRegExp)
		if err != nil {
			return nil, -1, fmt.Errorf("could not parse core info from %q: %v", line, err)
		}
		if ok {
			lastCore = val
			continue
		}

		ok, val, err = extractValue(line, nodeRegExp)
		if err != nil {
			return nil, -1, fmt.Errorf("could not parse node info from %q: %v", line, err)
		}
		if ok {
			lastNode = val
			continue
		}
	}

	nodeIdx, err := addNode(&nodes, lastNode)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to add node %d: %v", lastNode, err)
	}
	nodes[nodeIdx].AddThread(lastThread, lastCore)
	if numCores < 1 {
		return nil, numCores, fmt.Errorf("could not detect any cores")
	}
	for idx, node := range nodes {
		caches, err := sysinfo.GetCacheInfo(sysFs, node.Cores[0].Threads[0])
		if err != nil {
			klog.Errorf("failed to get cache information for node %d: %v", node.Id, err)
			continue
		}
		numThreadsPerCore := len(node.Cores[0].Threads)
		numThreadsPerNode := len(node.Cores) * numThreadsPerCore
		for _, cache := range caches {
			c := info.Cache{
				Size:  cache.Size,
				Level: cache.Level,
				Type:  cache.Type,
			}
			if cache.Cpus == numThreadsPerNode && cache.Level > 2 {
				// Add a node-level cache.
				nodes[idx].AddNodeCache(c)
			} else if cache.Cpus == numThreadsPerCore {
				// Add to each core.
				nodes[idx].AddPerCoreCache(c)
			}
			// Ignore unknown caches.
		}
	}
	return nodes, numCores, nil
}

func extractValue(s string, r *regexp.Regexp) (bool, int, error) {
	matches := r.FindSubmatch([]byte(s))
	if len(matches) == 2 {
		val, err := strconv.ParseInt(string(matches[1]), 10, 32)
		if err != nil {
			return false, -1, err
		}
		return true, int(val), nil
	}
	return false, -1, nil
}

// getUniqueMatchesCount returns number of unique matches in given argument using provided regular expression
func getUniqueMatchesCount(s string, r *regexp.Regexp) int {
	matches := r.FindAllString(s, -1)
	uniques := make(map[string]bool)
	for _, match := range matches {
		uniques[match] = true
	}
	return len(uniques)
}

func findNode(nodes []info.Node, id int) (bool, int) {
	for i, n := range nodes {
		if n.Id == id {
			return true, i
		}
	}
	return false, -1
}

func addNode(nodes *[]info.Node, id int) (int, error) {
	var idx int
	if id == -1 {
		// Some VMs don't fill topology data. Export single package.
		id = 0
	}

	ok, idx := findNode(*nodes, id)
	if !ok {
		// New node
		node := info.Node{Id: id}
		// Add per-node memory information.
		meminfo := fmt.Sprintf("/sys/devices/system/node/node%d/meminfo", id)
		out, err := ioutil.ReadFile(meminfo)
		// Ignore if per-node info is not available.
		if err == nil {
			m, err := parseCapacity(out, memoryCapacityRegexp)
			if err != nil {
				return -1, err
			}
			node.Memory = uint64(m)
		}
		// Look for per-node hugepages info using node id
		// Such as: /sys/devices/system/node/node%d/hugepages
		hugepagesDirectory := fmt.Sprintf("%s/node%d/hugepages/", nodePath, id)
		hugePagesInfo, err := GetHugePagesInfo(hugepagesDirectory)
		if err != nil {
			return -1, err
		}
		node.HugePages = hugePagesInfo

		*nodes = append(*nodes, node)
		idx = len(*nodes) - 1
	}
	return idx, nil
}

// s390/s390x changes
func getMachineArch() (string, error) {
	uname := unix.Utsname{}
	err := unix.Uname(&uname)
	if err != nil {
		return "", err
	}

	return string(uname.Machine[:]), nil
}

// arm32 chanes
func isArm32() bool {
	arch, err := getMachineArch()
	if err == nil {
		return strings.Contains(arch, "arm")
	}
	return false
}

// aarch64 changes
func isAArch64() bool {
	arch, err := getMachineArch()
	if err == nil {
		return strings.Contains(arch, "aarch64")
	}
	return false
}

// s390/s390x changes
func isSystemZ() bool {
	arch, err := getMachineArch()
	if err == nil {
		return strings.Contains(arch, "390")
	}
	return false
}

// riscv64 changes
func isRiscv64() bool {
	arch, err := getMachineArch()
	if err == nil {
		return strings.Contains(arch, "riscv64")
	}
	return false
}

// mips64 changes
func isMips64() bool {
	arch, err := getMachineArch()
	if err == nil {
		return strings.Contains(arch, "mips64")
	}
	return false
}

// s390/s390x changes
func getNumCores() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()

	if maxProcs < numCPU {
		return maxProcs
	}

	return numCPU
}
