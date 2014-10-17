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
}

func TestFileNotExist(t *testing.T) {
	cHints, err := getContainerHintsFromFile("/file_does_not_exist.json")
	if err != nil {
		t.Fatalf("getContainerHintsFromFile must not error for blank file: %s", err)
	}
	for _, container := range cHints.AllHosts {
		t.Logf("Container: %s", container)
	}
}
