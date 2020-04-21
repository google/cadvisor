// Copyright 2018 Google Inc. All Rights Reserved.
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

package libcontainer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/cadvisor/container"
	info "github.com/google/cadvisor/info/v1"
	"golang.org/x/sys/unix"

	"bytes"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"k8s.io/klog/v2"
)

type Handler struct {
	cgroupManager   cgroups.Manager
	rootFs          string
	pid             int
	includedMetrics container.MetricSet
	pidMetricsCache map[int]*info.CPUSchedstat
}

var whitelistedUlimits = [...]string{"max_open_files"}

func NewHandler(cgroupManager cgroups.Manager, rootFs string, pid int, includedMetrics container.MetricSet) *Handler {
	return &Handler{
		cgroupManager:   cgroupManager,
		rootFs:          rootFs,
		pid:             pid,
		includedMetrics: includedMetrics,
		pidMetricsCache: make(map[int]*info.CPUSchedstat),
	}
}

// Get cgroup and networking stats of the specified container
func (h *Handler) GetStats() (*info.ContainerStats, error) {
	cgroupStats, err := h.cgroupManager.GetStats()
	if err != nil {
		return nil, err
	}
	libcontainerStats := &libcontainer.Stats{
		CgroupStats: cgroupStats,
	}
	stats := newContainerStats(libcontainerStats, h.includedMetrics)

	if h.includedMetrics.Has(container.ProcessSchedulerMetrics) {
		pids, err := h.cgroupManager.GetAllPids()
		if err != nil {
			klog.V(4).Infof("Could not get PIDs for container %d: %v", h.pid, err)
		} else {
			stats.CPU.Schedstat, err = schedulerStatsFromProcs(h.rootFs, pids, h.pidMetricsCache)
			if err != nil {
				klog.V(4).Infof("Unable to get Process Scheduler Stats: %v", err)
			}
		}
	}

	// If we know the pid then get network stats from /proc/<pid>/net/dev
	if h.pid == 0 {
		return stats, nil
	}
	if h.includedMetrics.Has(container.NetworkUsageMetrics) {
		netStats, err := networkStatsFromProc(h.rootFs, h.pid)
		if err != nil {
			klog.V(4).Infof("Unable to get network stats from pid %d: %v", h.pid, err)
		} else {
			stats.Network.Interfaces = append(stats.Network.Interfaces, netStats...)
		}
	}
	if h.includedMetrics.Has(container.NetworkTCPUsageMetrics) {
		t, err := tcpStatsFromProc(h.rootFs, h.pid, "net/tcp")
		if err != nil {
			klog.V(4).Infof("Unable to get tcp stats from pid %d: %v", h.pid, err)
		} else {
			stats.Network.TCP = t
		}

		t6, err := tcpStatsFromProc(h.rootFs, h.pid, "net/tcp6")
		if err != nil {
			klog.V(4).Infof("Unable to get tcp6 stats from pid %d: %v", h.pid, err)
		} else {
			stats.Network.TCP6 = t6
		}

	}
	if h.includedMetrics.Has(container.NetworkAdvancedTCPUsageMetrics) {
		ta, err := advancedTCPStatsFromProc(h.rootFs, h.pid, "net/netstat", "net/snmp")
		if err != nil {
			klog.V(4).Infof("Unable to get advanced tcp stats from pid %d: %v", h.pid, err)
		} else {
			stats.Network.TCPAdvanced = ta
		}
	}
	if h.includedMetrics.Has(container.NetworkUDPUsageMetrics) {
		u, err := udpStatsFromProc(h.rootFs, h.pid, "net/udp")
		if err != nil {
			klog.V(4).Infof("Unable to get udp stats from pid %d: %v", h.pid, err)
		} else {
			stats.Network.UDP = u
		}

		u6, err := udpStatsFromProc(h.rootFs, h.pid, "net/udp6")
		if err != nil {
			klog.V(4).Infof("Unable to get udp6 stats from pid %d: %v", h.pid, err)
		} else {
			stats.Network.UPD6 = u6
		}
	}
	if h.includedMetrics.Has(container.ProcessMetrics) {
		paths := h.cgroupManager.GetPaths()
		path, ok := paths["cpu"]
		if !ok {
			klog.V(4).Infof("Could not find cgroups CPU for container %d", h.pid)
		} else {
			stats.Processes, err = processStatsFromProcs(h.rootFs, path, h.pid)
			if err != nil {
				klog.V(4).Infof("Unable to get Process Stats: %v", err)
			}
		}

		// if include processes metrics, just set threads metrics if exist, and has no relationship with cpu path
		setThreadsStats(cgroupStats, stats)
	}

	// For backwards compatibility.
	if len(stats.Network.Interfaces) > 0 {
		stats.Network.InterfaceStats = stats.Network.Interfaces[0]
	}

	return stats, nil
}

func parseUlimit(value string) (int64, error) {
	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		if strings.EqualFold(value, "unlimited") {
			// -1 implies unlimited except for priority and nice; man limits.conf
			num = -1
		} else {
			// Value is not a number or "unlimited"; return an error
			return 0, fmt.Errorf("unable to parse limit: %s", value)
		}
	}
	return num, nil
}

func isUlimitWhitelisted(name string) bool {
	for _, whitelist := range whitelistedUlimits {
		if name == whitelist {
			return true
		}
	}
	return false
}

func processLimitsFile(fileData string) []info.UlimitSpec {
	limits := strings.Split(fileData, "\n")
	ulimits := make([]info.UlimitSpec, 0, len(limits))
	for _, lim := range limits {
		// Skip any headers/footers
		if strings.HasPrefix(lim, "Max") {

			// Line format: Max open files            16384                16384                files
			fields := regexp.MustCompile(`[\s]{2,}`).Split(lim, -1)
			name := strings.Replace(strings.ToLower(strings.TrimSpace(fields[0])), " ", "_", -1)

			found := isUlimitWhitelisted(name)
			if !found {
				continue
			}

			soft := strings.TrimSpace(fields[1])
			soft_num, soft_err := parseUlimit(soft)

			hard := strings.TrimSpace(fields[2])
			hard_num, hard_err := parseUlimit(hard)

			// Omit metric if there were any parsing errors
			if soft_err == nil && hard_err == nil {
				ulimitSpec := info.UlimitSpec{
					Name:      name,
					SoftLimit: int64(soft_num),
					HardLimit: int64(hard_num),
				}
				ulimits = append(ulimits, ulimitSpec)
			}
		}
	}
	return ulimits
}

func processRootProcUlimits(rootFs string, rootPid int) []info.UlimitSpec {
	filePath := path.Join(rootFs, "/proc", strconv.Itoa(rootPid), "limits")
	out, err := ioutil.ReadFile(filePath)
	if err != nil {
		klog.V(4).Infof("error while listing directory %q to read ulimits: %v", filePath, err)
		return []info.UlimitSpec{}
	}
	return processLimitsFile(string(out))
}

func processStatsFromProcs(rootFs string, cgroupPath string, rootPid int) (info.ProcessStats, error) {
	var fdCount, socketCount uint64
	filePath := path.Join(cgroupPath, "cgroup.procs")
	out, err := ioutil.ReadFile(filePath)
	if err != nil {
		return info.ProcessStats{}, fmt.Errorf("couldn't open cpu cgroup procs file %v : %v", filePath, err)
	}

	pids := strings.Split(string(out), "\n")

	// EOL is also treated as a new line while reading "cgroup.procs" file with ioutil.ReadFile.
	// The last value is an empty string "". Ex: pids = ["22", "1223", ""]
	// Trim the last value
	if len(pids) != 0 && pids[len(pids)-1] == "" {
		pids = pids[:len(pids)-1]
	}

	for _, pid := range pids {
		dirPath := path.Join(rootFs, "/proc", pid, "fd")
		fds, err := ioutil.ReadDir(dirPath)
		if err != nil {
			klog.V(4).Infof("error while listing directory %q to measure fd count: %v", dirPath, err)
			continue
		}
		fdCount += uint64(len(fds))
		for _, fd := range fds {
			fdPath := path.Join(dirPath, fd.Name())
			linkName, err := os.Readlink(fdPath)
			if err != nil {
				klog.V(4).Infof("error while reading %q link: %v", fdPath, err)
				continue
			}
			if strings.HasPrefix(linkName, "socket") {
				socketCount++
			}
		}
	}
	ulimits := processRootProcUlimits(rootFs, rootPid)

	processStats := info.ProcessStats{
		ProcessCount: uint64(len(pids)),
		FdCount:      fdCount,
		SocketCount:  socketCount,
		Ulimits:      ulimits,
	}

	return processStats, nil
}

func schedulerStatsFromProcs(rootFs string, pids []int, pidMetricsCache map[int]*info.CPUSchedstat) (info.CPUSchedstat, error) {
	for _, pid := range pids {
		f, err := os.Open(path.Join(rootFs, "proc", strconv.Itoa(pid), "schedstat"))
		if err != nil {
			return info.CPUSchedstat{}, fmt.Errorf("couldn't open scheduler statistics for process %d: %v", pid, err)
		}
		defer f.Close()
		contents, err := ioutil.ReadAll(f)
		if err != nil {
			return info.CPUSchedstat{}, fmt.Errorf("couldn't read scheduler statistics for process %d: %v", pid, err)
		}
		rawMetrics := bytes.Split(bytes.TrimRight(contents, "\n"), []byte(" "))
		if len(rawMetrics) != 3 {
			return info.CPUSchedstat{}, fmt.Errorf("unexpected number of metrics in schedstat file for process %d", pid)
		}
		cacheEntry, ok := pidMetricsCache[pid]
		if !ok {
			cacheEntry = &info.CPUSchedstat{}
			pidMetricsCache[pid] = cacheEntry
		}
		for i, rawMetric := range rawMetrics {
			metric, err := strconv.ParseUint(string(rawMetric), 10, 64)
			if err != nil {
				return info.CPUSchedstat{}, fmt.Errorf("parsing error while reading scheduler statistics for process: %d: %v", pid, err)
			}
			switch i {
			case 0:
				cacheEntry.RunTime = metric
			case 1:
				cacheEntry.RunqueueTime = metric
			case 2:
				cacheEntry.RunPeriods = metric
			}
		}
	}
	schedstats := info.CPUSchedstat{}
	for _, v := range pidMetricsCache {
		schedstats.RunPeriods += v.RunPeriods
		schedstats.RunqueueTime += v.RunqueueTime
		schedstats.RunTime += v.RunTime
	}
	return schedstats, nil
}

func networkStatsFromProc(rootFs string, pid int) ([]info.InterfaceStats, error) {
	netStatsFile := path.Join(rootFs, "proc", strconv.Itoa(pid), "/net/dev")

	ifaceStats, err := scanInterfaceStats(netStatsFile)
	if err != nil {
		return []info.InterfaceStats{}, fmt.Errorf("couldn't read network stats: %v", err)
	}

	return ifaceStats, nil
}

var (
	ignoredDevicePrefixes = []string{"lo", "veth", "docker"}
)

func isIgnoredDevice(ifName string) bool {
	for _, prefix := range ignoredDevicePrefixes {
		if strings.HasPrefix(strings.ToLower(ifName), prefix) {
			return true
		}
	}
	return false
}

func scanInterfaceStats(netStatsFile string) ([]info.InterfaceStats, error) {
	file, err := os.Open(netStatsFile)
	if err != nil {
		return nil, fmt.Errorf("failure opening %s: %v", netStatsFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Discard header lines
	for i := 0; i < 2; i++ {
		if b := scanner.Scan(); !b {
			return nil, scanner.Err()
		}
	}

	stats := []info.InterfaceStats{}
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.Replace(line, ":", "", -1)

		fields := strings.Fields(line)
		// If the format of the  line is invalid then don't trust any of the stats
		// in this file.
		if len(fields) != 17 {
			return nil, fmt.Errorf("invalid interface stats line: %v", line)
		}

		devName := fields[0]
		if isIgnoredDevice(devName) {
			continue
		}

		i := info.InterfaceStats{
			Name: devName,
		}

		statFields := append(fields[1:5], fields[9:13]...)
		statPointers := []*uint64{
			&i.RxBytes, &i.RxPackets, &i.RxErrors, &i.RxDropped,
			&i.TxBytes, &i.TxPackets, &i.TxErrors, &i.TxDropped,
		}

		err := setInterfaceStatValues(statFields, statPointers)
		if err != nil {
			return nil, fmt.Errorf("cannot parse interface stats (%v): %v", err, line)
		}

		stats = append(stats, i)
	}

	return stats, nil
}

func setInterfaceStatValues(fields []string, pointers []*uint64) error {
	for i, v := range fields {
		val, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return err
		}
		*pointers[i] = val
	}
	return nil
}

func tcpStatsFromProc(rootFs string, pid int, file string) (info.TCPStat, error) {
	tcpStatsFile := path.Join(rootFs, "proc", strconv.Itoa(pid), file)

	tcpStats, err := scanTCPStats(tcpStatsFile)
	if err != nil {
		return tcpStats, fmt.Errorf("couldn't read tcp stats: %v", err)
	}

	return tcpStats, nil
}

func advancedTCPStatsFromProc(rootFs string, pid int, file1, file2 string) (info.TCPAdvancedStat, error) {
	var advancedStats info.TCPAdvancedStat
	var err error

	netstatFile := path.Join(rootFs, "proc", strconv.Itoa(pid), file1)
	err = scanAdvancedTCPStats(&advancedStats, netstatFile)
	if err != nil {
		return advancedStats, err
	}

	snmpFile := path.Join(rootFs, "proc", strconv.Itoa(pid), file2)
	err = scanAdvancedTCPStats(&advancedStats, snmpFile)
	if err != nil {
		return advancedStats, err
	}

	return advancedStats, nil
}

func scanAdvancedTCPStats(advancedStats *info.TCPAdvancedStat, advancedTCPStatsFile string) error {
	data, err := ioutil.ReadFile(advancedTCPStatsFile)
	if err != nil {
		return fmt.Errorf("failure opening %s: %v", advancedTCPStatsFile, err)
	}

	reader := strings.NewReader(string(data))
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	advancedTCPStats := make(map[string]interface{})
	for scanner.Scan() {
		nameParts := strings.Split(scanner.Text(), " ")
		scanner.Scan()
		valueParts := strings.Split(scanner.Text(), " ")
		// Remove trailing :. and ignore non-tcp
		protocol := nameParts[0][:len(nameParts[0])-1]
		if protocol != "TcpExt" && protocol != "Tcp" {
			continue
		}
		if len(nameParts) != len(valueParts) {
			return fmt.Errorf("mismatch field count mismatch in %s: %s",
				advancedTCPStatsFile, protocol)
		}
		for i := 1; i < len(nameParts); i++ {
			if strings.Contains(valueParts[i], "-") {
				vInt64, err := strconv.ParseInt(valueParts[i], 10, 64)
				if err != nil {
					return fmt.Errorf("decode value: %s to int64 error: %s", valueParts[i], err)
				}
				advancedTCPStats[nameParts[i]] = vInt64
			} else {
				vUint64, err := strconv.ParseUint(valueParts[i], 10, 64)
				if err != nil {
					return fmt.Errorf("decode value: %s to uint64 error: %s", valueParts[i], err)
				}
				advancedTCPStats[nameParts[i]] = vUint64
			}
		}
	}

	b, err := json.Marshal(advancedTCPStats)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, advancedStats)
	if err != nil {
		return err
	}

	return scanner.Err()

}

func scanTCPStats(tcpStatsFile string) (info.TCPStat, error) {

	var stats info.TCPStat

	data, err := ioutil.ReadFile(tcpStatsFile)
	if err != nil {
		return stats, fmt.Errorf("failure opening %s: %v", tcpStatsFile, err)
	}

	tcpStateMap := map[string]uint64{
		"01": 0, //ESTABLISHED
		"02": 0, //SYN_SENT
		"03": 0, //SYN_RECV
		"04": 0, //FIN_WAIT1
		"05": 0, //FIN_WAIT2
		"06": 0, //TIME_WAIT
		"07": 0, //CLOSE
		"08": 0, //CLOSE_WAIT
		"09": 0, //LAST_ACK
		"0A": 0, //LISTEN
		"0B": 0, //CLOSING
	}

	reader := strings.NewReader(string(data))
	scanner := bufio.NewScanner(reader)

	scanner.Split(bufio.ScanLines)

	// Discard header line
	if b := scanner.Scan(); !b {
		return stats, scanner.Err()
	}

	for scanner.Scan() {
		line := scanner.Text()

		state := strings.Fields(line)
		// TCP state is the 4th field.
		// Format: sl local_address rem_address st tx_queue rx_queue tr tm->when retrnsmt  uid timeout inode
		tcpState := state[3]
		_, ok := tcpStateMap[tcpState]
		if !ok {
			return stats, fmt.Errorf("invalid TCP stats line: %v", line)
		}
		tcpStateMap[tcpState]++
	}

	stats = info.TCPStat{
		Established: tcpStateMap["01"],
		SynSent:     tcpStateMap["02"],
		SynRecv:     tcpStateMap["03"],
		FinWait1:    tcpStateMap["04"],
		FinWait2:    tcpStateMap["05"],
		TimeWait:    tcpStateMap["06"],
		Close:       tcpStateMap["07"],
		CloseWait:   tcpStateMap["08"],
		LastAck:     tcpStateMap["09"],
		Listen:      tcpStateMap["0A"],
		Closing:     tcpStateMap["0B"],
	}

	return stats, nil
}

func udpStatsFromProc(rootFs string, pid int, file string) (info.UDPStat, error) {
	var err error
	var udpStats info.UDPStat

	udpStatsFile := path.Join(rootFs, "proc", strconv.Itoa(pid), file)

	r, err := os.Open(udpStatsFile)
	if err != nil {
		return udpStats, fmt.Errorf("failure opening %s: %v", udpStatsFile, err)
	}

	udpStats, err = scanUDPStats(r)
	if err != nil {
		return udpStats, fmt.Errorf("couldn't read udp stats: %v", err)
	}

	return udpStats, nil
}

func scanUDPStats(r io.Reader) (info.UDPStat, error) {
	var stats info.UDPStat

	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)

	// Discard header line
	if b := scanner.Scan(); !b {
		return stats, scanner.Err()
	}

	listening := uint64(0)
	dropped := uint64(0)
	rxQueued := uint64(0)
	txQueued := uint64(0)

	for scanner.Scan() {
		line := scanner.Text()
		// Format: sl local_address rem_address st tx_queue rx_queue tr tm->when retrnsmt  uid timeout inode ref pointer drops

		listening++

		fs := strings.Fields(line)
		if len(fs) != 13 {
			continue
		}

		rx, tx := uint64(0), uint64(0)
		fmt.Sscanf(fs[4], "%X:%X", &rx, &tx)
		rxQueued += rx
		txQueued += tx

		d, err := strconv.Atoi(string(fs[12]))
		if err != nil {
			continue
		}
		dropped += uint64(d)
	}

	stats = info.UDPStat{
		Listen:   listening,
		Dropped:  dropped,
		RxQueued: rxQueued,
		TxQueued: txQueued,
	}

	return stats, nil
}

func (h *Handler) GetProcesses() ([]int, error) {
	pids, err := h.cgroupManager.GetPids()
	if err != nil {
		return nil, err
	}
	return pids, nil
}

func minUint32(x, y uint32) uint32 {
	if x < y {
		return x
	}
	return y
}

// var to allow unit tests to stub it out
var numCpusFunc = getNumberOnlineCPUs

// Convert libcontainer stats to info.ContainerStats.
func setCPUStats(s *cgroups.Stats, ret *info.ContainerStats, withPerCPU bool) {
	ret.CPU.Usage.User = s.CpuStats.CpuUsage.UsageInUsermode
	ret.CPU.Usage.System = s.CpuStats.CpuUsage.UsageInKernelmode
	ret.CPU.Usage.Total = s.CpuStats.CpuUsage.TotalUsage
	ret.CPU.CFS.Periods = s.CpuStats.ThrottlingData.Periods
	ret.CPU.CFS.ThrottledPeriods = s.CpuStats.ThrottlingData.ThrottledPeriods
	ret.CPU.CFS.ThrottledTime = s.CpuStats.ThrottlingData.ThrottledTime

	if !withPerCPU {
		return
	}
	if len(s.CpuStats.CpuUsage.PercpuUsage) == 0 {
		// libcontainer's 'GetStats' can leave 'PercpuUsage' nil if it skipped the
		// cpuacct subsystem.
		return
	}

	numPossible := uint32(len(s.CpuStats.CpuUsage.PercpuUsage))
	// Note that as of https://patchwork.kernel.org/patch/8607101/ (kernel v4.7),
	// the percpu usage information includes extra zero values for all additional
	// possible CPUs. This is to allow statistic collection after CPU-hotplug.
	// We intentionally ignore these extra zeroes.
	numActual, err := numCpusFunc()
	if err != nil {
		klog.Errorf("unable to determine number of actual cpus; defaulting to maximum possible number: errno %v", err)
		numActual = numPossible
	}
	if numActual > numPossible {
		// The real number of cores should never be greater than the number of
		// datapoints reported in cpu usage.
		klog.Errorf("PercpuUsage had %v cpus, but the actual number is %v; ignoring extra CPUs", numPossible, numActual)
	}
	numActual = minUint32(numPossible, numActual)
	ret.CPU.Usage.PerCPU = make([]uint64, numActual)

	for i := uint32(0); i < numActual; i++ {
		ret.CPU.Usage.PerCPU[i] = s.CpuStats.CpuUsage.PercpuUsage[i]
	}

}

func getNumberOnlineCPUs() (uint32, error) {
	var availableCPUs unix.CPUSet
	if err := unix.SchedGetaffinity(0, &availableCPUs); err != nil {
		return 0, err
	}
	return uint32(availableCPUs.Count()), nil
}

func setDiskIoStats(s *cgroups.Stats, ret *info.ContainerStats) {
	ret.DiskIo.IoServiceBytes = DiskStatsCopy(s.BlkioStats.IoServiceBytesRecursive)
	ret.DiskIo.IoServiced = DiskStatsCopy(s.BlkioStats.IoServicedRecursive)
	ret.DiskIo.IoQueued = DiskStatsCopy(s.BlkioStats.IoQueuedRecursive)
	ret.DiskIo.Sectors = DiskStatsCopy(s.BlkioStats.SectorsRecursive)
	ret.DiskIo.IoServiceTime = DiskStatsCopy(s.BlkioStats.IoServiceTimeRecursive)
	ret.DiskIo.IoWaitTime = DiskStatsCopy(s.BlkioStats.IoWaitTimeRecursive)
	ret.DiskIo.IoMerged = DiskStatsCopy(s.BlkioStats.IoMergedRecursive)
	ret.DiskIo.IoTime = DiskStatsCopy(s.BlkioStats.IoTimeRecursive)
}

func setMemoryStats(s *cgroups.Stats, ret *info.ContainerStats) {
	ret.Memory.Usage = s.MemoryStats.Usage.Usage
	ret.Memory.MaxUsage = s.MemoryStats.Usage.MaxUsage
	ret.Memory.Failcnt = s.MemoryStats.Usage.Failcnt

	if s.MemoryStats.UseHierarchy {
		ret.Memory.Cache = s.MemoryStats.Stats["total_cache"]
		ret.Memory.RSS = s.MemoryStats.Stats["total_rss"]
		ret.Memory.Swap = s.MemoryStats.Stats["total_swap"]
		ret.Memory.MappedFile = s.MemoryStats.Stats["total_mapped_file"]
	} else {
		ret.Memory.Cache = s.MemoryStats.Stats["cache"]
		ret.Memory.RSS = s.MemoryStats.Stats["rss"]
		ret.Memory.Swap = s.MemoryStats.Stats["swap"]
		ret.Memory.MappedFile = s.MemoryStats.Stats["mapped_file"]
	}
	if v, ok := s.MemoryStats.Stats["pgfault"]; ok {
		ret.Memory.ContainerData.Pgfault = v
		ret.Memory.HierarchicalData.Pgfault = v
	}
	if v, ok := s.MemoryStats.Stats["pgmajfault"]; ok {
		ret.Memory.ContainerData.Pgmajfault = v
		ret.Memory.HierarchicalData.Pgmajfault = v
	}

	workingSet := ret.Memory.Usage
	if v, ok := s.MemoryStats.Stats["total_inactive_file"]; ok {
		if workingSet < v {
			workingSet = 0
		} else {
			workingSet -= v
		}
	}
	ret.Memory.WorkingSet = workingSet
}

func setHugepageStats(s *cgroups.Stats, ret *info.ContainerStats) {
	ret.Hugetlb = make(map[string]info.HugetlbStats)
	for k, v := range s.HugetlbStats {
		ret.Hugetlb[k] = info.HugetlbStats{
			Usage:    v.Usage,
			MaxUsage: v.MaxUsage,
			Failcnt:  v.Failcnt,
		}
	}
}

func setNetworkStats(libcontainerStats *libcontainer.Stats, ret *info.ContainerStats) {
	ret.Network.Interfaces = make([]info.InterfaceStats, len(libcontainerStats.Interfaces))
	for i := range libcontainerStats.Interfaces {
		ret.Network.Interfaces[i] = info.InterfaceStats{
			Name:      libcontainerStats.Interfaces[i].Name,
			RxBytes:   libcontainerStats.Interfaces[i].RxBytes,
			RxPackets: libcontainerStats.Interfaces[i].RxPackets,
			RxErrors:  libcontainerStats.Interfaces[i].RxErrors,
			RxDropped: libcontainerStats.Interfaces[i].RxDropped,
			TxBytes:   libcontainerStats.Interfaces[i].TxBytes,
			TxPackets: libcontainerStats.Interfaces[i].TxPackets,
			TxErrors:  libcontainerStats.Interfaces[i].TxErrors,
			TxDropped: libcontainerStats.Interfaces[i].TxDropped,
		}
	}

	// Add to base struct for backwards compatibility.
	if len(ret.Network.Interfaces) > 0 {
		ret.Network.InterfaceStats = ret.Network.Interfaces[0]
	}
}

// read from pids path not cpu
func setThreadsStats(s *cgroups.Stats, ret *info.ContainerStats) {
	if s != nil {
		ret.Processes.ThreadsCurrent = s.PidsStats.Current
		ret.Processes.ThreadsMax = s.PidsStats.Limit
	}

}

func newContainerStats(libcontainerStats *libcontainer.Stats, includedMetrics container.MetricSet) *info.ContainerStats {
	ret := &info.ContainerStats{
		Timestamp: time.Now(),
	}

	if s := libcontainerStats.CgroupStats; s != nil {
		setCPUStats(s, ret, includedMetrics.Has(container.PerCPUUsageMetrics))
		if includedMetrics.Has(container.DiskIOMetrics) {
			setDiskIoStats(s, ret)
		}
		setMemoryStats(s, ret)
		if includedMetrics.Has(container.HugetlbUsageMetrics) {
			setHugepageStats(s, ret)
		}
	}
	if len(libcontainerStats.Interfaces) > 0 {
		setNetworkStats(libcontainerStats, ret)
	}
	return ret
}
