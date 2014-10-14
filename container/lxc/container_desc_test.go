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
	if cDesc.All_hosts[0].Network_interface.VethHost != "veth24031eth1" &&
		cDesc.All_hosts[0].Network_interface.VethChild != "eth1" {
		t.Errorf("Cannot find network interface in %s", cDesc)
	}
}
