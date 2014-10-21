package raw

import (
	"testing"
)

func TestGetContainerHintsFromFile(t *testing.T) {
	cHints, err := getContainerHintsFromFile("test_resources/container_hints.json")

	if err != nil {
		t.Fatalf("Error in unmarshalling: %s", err)
	}

	if cHints.AllHosts[0].NetworkInterface.VethHost != "veth24031eth1" &&
		cHints.AllHosts[0].NetworkInterface.VethChild != "eth1" {
		t.Errorf("Cannot find network interface in %s", cHints)
	}

	correctMountDirs := [...]string{
		"/var/run/nm-sdc1",
		"/var/run/nm-sdb3",
		"/var/run/nm-sda3",
		"/var/run/netns/root",
		"/var/run/openvswitch/db.sock",
	}

	if len(cHints.AllHosts[0].Mounts) == 0 {
		t.Errorf("Cannot find any mounts")
	}

	for i, mountDir := range cHints.AllHosts[0].Mounts {
		if correctMountDirs[i] != mountDir.HostDir {
			t.Errorf("Cannot find mount %s in %s", mountDir.HostDir, cHints)
		}
	}
}

func TestFileNotExist(t *testing.T) {
	_, err := getContainerHintsFromFile("/file_does_not_exist.json")
	if err != nil {
		t.Fatalf("getContainerHintsFromFile must not error for blank file: %s", err)
	}
}
