package api

import (
	"testing"
	"time"

	"github.com/google/cadvisor/info"
	"github.com/stretchr/testify/assert"
)

// Checks that expected and actual are within delta of each other.
func inDelta(t *testing.T, expected, actual, delta uint64, description string) {
	var diff uint64
	if expected > actual {
		diff = expected - actual
	} else {
		diff = actual - expected
	}
	if diff > delta {
		t.Errorf("%s (%d and %d) are not within %d of each other", description, expected, actual, delta)
	}
}

// Checks that CPU stats are valid.
func checkCpuStats(t *testing.T, stat info.CpuStats) {
	assert := assert.New(t)

	assert.NotEqual(0, stat.Usage.Total, "Total CPU usage should not be zero")
	assert.NotEmpty(stat.Usage.PerCpu, "Per-core usage should not be empty")
	totalUsage := uint64(0)
	for _, usage := range stat.Usage.PerCpu {
		totalUsage += usage
	}
	inDelta(t, stat.Usage.Total, totalUsage, uint64((5 * time.Millisecond).Nanoseconds()), "Per-core CPU usage")
	inDelta(t, stat.Usage.Total, stat.Usage.User+stat.Usage.System, uint64((25 * time.Millisecond).Nanoseconds()), "User + system CPU usage")
	assert.Equal(0, stat.Load, "Non-zero load is unexpected as it is currently unset. Do we need to update the test?")
}

func checkMemoryStats(t *testing.T, stat info.MemoryStats) {
	assert := assert.New(t)

	assert.NotEqual(0, stat.Usage, "Memory usage should not be zero")
	assert.NotEqual(0, stat.WorkingSet, "Memory working set should not be zero")
	if stat.WorkingSet > stat.Usage {
		t.Errorf("Memory working set (%d) should be at most equal to memory usage (%d)", stat.WorkingSet, stat.Usage)
	}
	// TODO(vmarmol): Add checks for ContainerData and HierarchicalData
}
