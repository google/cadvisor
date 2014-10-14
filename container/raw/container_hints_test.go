package raw

import (
	"testing"
)

func TestUnmarshal(t *testing.T) {
	cHints, err := getContainerHintsFromFile("test_resources/container_hints.json")

	if err != nil {
		t.Fatalf("Error in unmarshalling: %s", err)
	}

	if cHints.AllHosts[0].NetworkInterface.VethHost != "veth24031eth1" &&
		cHints.AllHosts[0].NetworkInterface.VethChild != "eth1" {
		t.Errorf("Cannot find network interface in %s", cHints)
	}
}
