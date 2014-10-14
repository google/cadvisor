package raw

import (
	"testing"
)

func TestUnmarshal(t *testing.T) {
	cDesc, err := Unmarshal("test_resources/cdesc.json")

	if err != nil {
		t.Fatalf("Error in unmarshalling: %s", err)
	}

	if cDesc.All_hosts[0].NetworkInterface.VethHost != "veth24031eth1" &&
		cDesc.All_hosts[0].NetworkInterface.VethChild != "eth1" {
		t.Errorf("Cannot find network interface in %s", cDesc)
	}
}
