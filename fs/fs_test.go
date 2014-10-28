package fs

import (
	"testing"
)

func TestGetDiskStatsMap(t *testing.T) {
	diskStatsMap, err := getDiskStatsMap("test_resources/diskstats")
	if err != nil {
		t.Errorf("Error calling getDiskStatMap %s", err)
	}
	if len(diskStatsMap) != 20 {
		t.Errorf("diskStatsMap %s not valid", diskStatsMap)
	}
	keySet := map[string]string{
		"/dev/sdb1": "/dev/sdb1",
		"/dev/sdb2": "/dev/sdb2",
		"/dev/sda1": "/dev/sda1",
		"/dev/sda2": "/dev/sda2",
		"/dev/sdc1": "/dev/sdc1",
		"/dev/sdc2": "/dev/sdc2",
		"/dev/sdc3": "/dev/sdc3",
		"/dev/sdc4": "/dev/sdc4",
		"/dev/sdd1": "/dev/sdd1",
		"/dev/sdd2": "/dev/sdd2",
		"/dev/sdd3": "/dev/sdd3",
		"/dev/sdd4": "/dev/sdd4",
		"/dev/sde1": "/dev/sde1",
		"/dev/sde2": "/dev/sde2",
		"/dev/sdf1": "/dev/sdf1",
		"/dev/sdf2": "/dev/sdf2",
		"/dev/sdg1": "/dev/sdg1",
		"/dev/sdg2": "/dev/sdg2",
		"/dev/sdh1": "/dev/sdh1",
		"/dev/sdh2": "/dev/sdh2",
	}

	for device := range diskStatsMap {
		if _, ok := keySet[device]; !ok {
			t.Errorf("Cannot find device %s", device)
		}
		delete(keySet, device)
	}
	if len(keySet) != 0 {
		t.Errorf("diskStatsMap %s contains illegal keys %s", diskStatsMap, keySet)
	}
}

func TestFileNotExist(t *testing.T) {
	_, err := getDiskStatsMap("/file_does_not_exist")
	if err != nil {
		t.Fatalf("getDiskStatsMap must not error for absent file: %s", err)
	}
}
