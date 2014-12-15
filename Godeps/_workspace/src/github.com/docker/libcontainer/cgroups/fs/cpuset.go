package fs

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/docker/libcontainer/cgroups"
)

type CpusetGroup struct {
}

func (s *CpusetGroup) Set(d *data) error {
	dir, err := d.path("cpuset")
	if err != nil {
		return err
	}
	return s.SetDir(dir, d.c.CpusetCpus, d.c.CpusetMems, d.pid)
}

func (s *CpusetGroup) Remove(d *data) error {
	return removePath(d.path("cpuset"))
}

func (s *CpusetGroup) GetStats(path string, stats *cgroups.Stats) error {
	return nil
}

func (s *CpusetGroup) SetDir(dir, cpus string, mems string, pid int) error {
	if err := s.ensureParent(dir); err != nil {
		return err
	}

	// because we are not using d.join we need to place the pid into the procs file
	// unlike the other subsystems
	if err := writeFile(dir, "cgroup.procs", strconv.Itoa(pid)); err != nil {
		return err
	}

	// If we don't use --cpuset-xxx, the default value inherit from parent cgroup
	// is set in s.ensureParent, otherwise, use the value we set
	if cpus != "" {
		if err := writeFile(dir, "cpuset.cpus", cpus); err != nil {
			return err
		}
	}
	if mems != "" {
		if err := writeFile(dir, "cpuset.mems", mems); err != nil {
			return err
		}
	}

	return nil
}

func (s *CpusetGroup) getSubsystemSettings(parent string) (cpus []byte, mems []byte, err error) {
	if cpus, err = ioutil.ReadFile(filepath.Join(parent, "cpuset.cpus")); err != nil {
		return
	}
	if mems, err = ioutil.ReadFile(filepath.Join(parent, "cpuset.mems")); err != nil {
		return
	}
	return cpus, mems, nil
}

// ensureParent ensures that the parent directory of current is created
// with the proper cpus and mems files copied from it's parent if the values
// are a file with a new line char
func (s *CpusetGroup) ensureParent(current string) error {
	parent := filepath.Dir(current)

	if _, err := os.Stat(parent); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		if err := s.ensureParent(parent); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(current, 0755); err != nil && !os.IsExist(err) {
		return err
	}
	return s.copyIfNeeded(current, parent)
}

// copyIfNeeded copies the cpuset.cpus and cpuset.mems from the parent
// directory to the current directory if the file's contents are 0
func (s *CpusetGroup) copyIfNeeded(current, parent string) error {
	var (
		err                      error
		currentCpus, currentMems []byte
		parentCpus, parentMems   []byte
	)

	if currentCpus, currentMems, err = s.getSubsystemSettings(current); err != nil {
		return err
	}
	if parentCpus, parentMems, err = s.getSubsystemSettings(parent); err != nil {
		return err
	}

	if s.isEmpty(currentCpus) {
		if err := writeFile(current, "cpuset.cpus", string(parentCpus)); err != nil {
			return err
		}
	}
	if s.isEmpty(currentMems) {
		if err := writeFile(current, "cpuset.mems", string(parentMems)); err != nil {
			return err
		}
	}
	return nil
}

func (s *CpusetGroup) isEmpty(b []byte) bool {
	return len(bytes.Trim(b, "\n")) == 0
}
