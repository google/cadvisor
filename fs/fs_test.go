package fs

import (
	"io/ioutil"
	"reflect"
	"testing"
)

func TestGetDiskStatsMap(t *testing.T) {
	dat, err := ioutil.ReadFile("test_resources/diskstats")
	diskStats, err := getDiskStats(string(dat))
	if err != nil {
		t.Errorf("Error calling getDiskStats %s", err)
	}

	correctDiskStats := DiskStats{
		ReadsMerged:    13.0,
		WritesMerged:   33.0,
		ReadsIssued:    0.89,
		WritesIssued:   0.94,
		SectorsRead:    37.01,
		SectorsWritten: 79.43,
		AvgRequestSize: 63.84,
		AvgQueueLen:    3,
		AvgWaitTime:    5,
		AvgServiceTime: 6,
		PercentUtil:    5,
	}

	if !reflect.DeepEqual(diskStats, correctDiskStats) {
		t.Errorf("diskStats %s not valid", diskStats)
	}
}
