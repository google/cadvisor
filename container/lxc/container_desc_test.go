package lxc

import (
	"testing"
)

func TestUnmarshal(t *testing.T) {
	cDesc, err := Unmarshal("test_resources/cdesc.json")
	if err != nil {
		t.Fatalf("Error in unmarshalling: %s", err)
	}

	t.Logf("Cdesc: %s", cDesc)
	if cDesc.All_hosts[0].Network_interface != "veth7ASIQc" {
		t.Errorf("Cannot find network interface in %s", cDesc)
	}
}
