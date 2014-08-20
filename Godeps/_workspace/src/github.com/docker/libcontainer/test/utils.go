package libcontainer

import (
	"github.com/docker/libcontainer"
	"github.com/docker/libcontainer/security/capabilities"
)

func GetAllCapabilities() []string {
	return capabilities.GetAllCapabilities()
}

func DropBoundingSet(container *libcontainer.Config) error {
	return capabilities.DropBoundingSet(container.Capabilities)
}

func DropCapabilities(container *libcontainer.Config) error {
	return capabilities.DropCapabilities(container.Capabilities)
}
