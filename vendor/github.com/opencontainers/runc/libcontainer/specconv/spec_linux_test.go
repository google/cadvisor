// +build linux

package specconv

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/configs/validate"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func TestCreateCommandHookTimeout(t *testing.T) {
	timeout := 3600
	hook := specs.Hook{
		Path:    "/some/hook/path",
		Args:    []string{"--some", "thing"},
		Env:     []string{"SOME=value"},
		Timeout: &timeout,
	}
	command := createCommandHook(hook)
	timeoutStr := command.Timeout.String()
	if timeoutStr != "1h0m0s" {
		t.Errorf("Expected the Timeout to be 1h0m0s, got: %s", timeoutStr)
	}
}

func TestCreateHooks(t *testing.T) {
	rspec := &specs.Spec{
		Hooks: &specs.Hooks{
			Prestart: []specs.Hook{
				{
					Path: "/some/hook/path",
				},
				{
					Path: "/some/hook2/path",
					Args: []string{"--some", "thing"},
				},
			},
			Poststart: []specs.Hook{
				{
					Path: "/some/hook/path",
					Args: []string{"--some", "thing"},
					Env:  []string{"SOME=value"},
				},
				{
					Path: "/some/hook2/path",
				},
				{
					Path: "/some/hook3/path",
				},
			},
			Poststop: []specs.Hook{
				{
					Path: "/some/hook/path",
					Args: []string{"--some", "thing"},
					Env:  []string{"SOME=value"},
				},
				{
					Path: "/some/hook2/path",
				},
				{
					Path: "/some/hook3/path",
				},
				{
					Path: "/some/hook4/path",
					Args: []string{"--some", "thing"},
				},
			},
		},
	}
	conf := &configs.Config{}
	createHooks(rspec, conf)

	prestart := conf.Hooks.Prestart

	if len(prestart) != 2 {
		t.Error("Expected 2 Prestart hooks")
	}

	poststart := conf.Hooks.Poststart

	if len(poststart) != 3 {
		t.Error("Expected 3 Poststart hooks")
	}

	poststop := conf.Hooks.Poststop

	if len(poststop) != 4 {
		t.Error("Expected 4 Poststop hooks")
	}

}
func TestSetupSeccomp(t *testing.T) {
	conf := &specs.LinuxSeccomp{
		DefaultAction: "SCMP_ACT_ERRNO",
		Architectures: []specs.Arch{specs.ArchX86_64, specs.ArchARM},
		Syscalls: []specs.LinuxSyscall{
			{
				Names:  []string{"clone"},
				Action: "SCMP_ACT_ALLOW",
				Args: []specs.LinuxSeccompArg{
					{
						Index:    0,
						Value:    unix.CLONE_NEWNS | unix.CLONE_NEWUTS | unix.CLONE_NEWIPC | unix.CLONE_NEWUSER | unix.CLONE_NEWPID | unix.CLONE_NEWNET | unix.CLONE_NEWCGROUP,
						ValueTwo: 0,
						Op:       "SCMP_CMP_MASKED_EQ",
					},
				},
			},
			{
				Names: []string{
					"select",
					"semctl",
					"semget",
					"semop",
					"semtimedop",
					"send",
					"sendfile",
				},
				Action: "SCMP_ACT_ALLOW",
			},
		},
	}
	seccomp, err := SetupSeccomp(conf)

	if err != nil {
		t.Errorf("Couldn't create Seccomp config: %v", err)
	}

	if seccomp.DefaultAction != 2 { // SCMP_ACT_ERRNO
		t.Error("Wrong conversion for DefaultAction")
	}

	if len(seccomp.Architectures) != 2 {
		t.Error("Wrong number of architectures")
	}

	if seccomp.Architectures[0] != "amd64" || seccomp.Architectures[1] != "arm" {
		t.Error("Expected architectures are not found")
	}

	calls := seccomp.Syscalls

	callsLength := len(calls)
	if callsLength != 8 {
		t.Errorf("Expected 8 syscalls, got :%d", callsLength)
	}

	for i, call := range calls {
		if i == 0 {
			expectedCloneSyscallArgs := configs.Arg{
				Index:    0,
				Op:       7, // SCMP_CMP_MASKED_EQ
				Value:    unix.CLONE_NEWNS | unix.CLONE_NEWUTS | unix.CLONE_NEWIPC | unix.CLONE_NEWUSER | unix.CLONE_NEWPID | unix.CLONE_NEWNET | unix.CLONE_NEWCGROUP,
				ValueTwo: 0,
			}
			if expectedCloneSyscallArgs != *call.Args[0] {
				t.Errorf("Wrong arguments conversion for the clone syscall under test")
			}
		}
		if call.Action != 4 {
			t.Error("Wrong conversion for the clone syscall action")
		}

	}

}

func TestLinuxCgroupWithMemoryResource(t *testing.T) {
	cgroupsPath := "/user/cgroups/path/id"

	spec := &specs.Spec{}
	devices := []specs.LinuxDeviceCgroup{
		{
			Allow:  false,
			Access: "rwm",
		},
	}

	limit := int64(100)
	reservation := int64(50)
	swap := int64(20)
	kernel := int64(40)
	kernelTCP := int64(45)
	swappiness := uint64(1)
	swappinessPtr := &swappiness
	disableOOMKiller := true
	resources := &specs.LinuxResources{
		Devices: devices,
		Memory: &specs.LinuxMemory{
			Limit:            &limit,
			Reservation:      &reservation,
			Swap:             &swap,
			Kernel:           &kernel,
			KernelTCP:        &kernelTCP,
			Swappiness:       swappinessPtr,
			DisableOOMKiller: &disableOOMKiller,
		},
	}
	spec.Linux = &specs.Linux{
		CgroupsPath: cgroupsPath,
		Resources:   resources,
	}

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: false,
		Spec:             spec,
	}

	cgroup, err := createCgroupConfig(opts)
	if err != nil {
		t.Errorf("Couldn't create Cgroup config: %v", err)
	}

	if cgroup.Path != cgroupsPath {
		t.Errorf("Wrong cgroupsPath, expected '%s' got '%s'", cgroupsPath, cgroup.Path)
	}
	if cgroup.Resources.Memory != limit {
		t.Errorf("Expected to have %d as memory limit, got %d", limit, cgroup.Resources.Memory)
	}
	if cgroup.Resources.MemoryReservation != reservation {
		t.Errorf("Expected to have %d as memory reservation, got %d", reservation, cgroup.Resources.MemoryReservation)
	}
	if cgroup.Resources.MemorySwap != swap {
		t.Errorf("Expected to have %d as swap, got %d", swap, cgroup.Resources.MemorySwap)
	}
	if cgroup.Resources.KernelMemory != kernel {
		t.Errorf("Expected to have %d as Kernel Memory, got %d", kernel, cgroup.Resources.KernelMemory)
	}
	if cgroup.Resources.KernelMemoryTCP != kernelTCP {
		t.Errorf("Expected to have %d as TCP Kernel Memory, got %d", kernelTCP, cgroup.Resources.KernelMemoryTCP)
	}
	if cgroup.Resources.MemorySwappiness != swappinessPtr {
		t.Errorf("Expected to have %d as memory swappiness, got %d", swappinessPtr, cgroup.Resources.MemorySwappiness)
	}
	if cgroup.Resources.OomKillDisable != disableOOMKiller {
		t.Errorf("The OOMKiller should be enabled")
	}
}

func TestLinuxCgroupSystemd(t *testing.T) {
	cgroupsPath := "parent:scopeprefix:name"

	spec := &specs.Spec{}
	spec.Linux = &specs.Linux{
		CgroupsPath: cgroupsPath,
	}

	opts := &CreateOpts{
		UseSystemdCgroup: true,
		Spec:             spec,
	}

	cgroup, err := createCgroupConfig(opts)

	if err != nil {
		t.Errorf("Couldn't create Cgroup config: %v", err)
	}

	expectedParent := "parent"
	if cgroup.Parent != expectedParent {
		t.Errorf("Expected to have %s as Parent instead of %s", expectedParent, cgroup.Parent)
	}

	expectedScopePrefix := "scopeprefix"
	if cgroup.ScopePrefix != expectedScopePrefix {
		t.Errorf("Expected to have %s as ScopePrefix instead of %s", expectedScopePrefix, cgroup.ScopePrefix)
	}

	expectedName := "name"
	if cgroup.Name != expectedName {
		t.Errorf("Expected to have %s as Name instead of %s", expectedName, cgroup.Name)
	}
}

func TestLinuxCgroupSystemdWithEmptyPath(t *testing.T) {
	cgroupsPath := ""

	spec := &specs.Spec{}
	spec.Linux = &specs.Linux{
		CgroupsPath: cgroupsPath,
	}

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: true,
		Spec:             spec,
	}

	cgroup, err := createCgroupConfig(opts)

	if err != nil {
		t.Errorf("Couldn't create Cgroup config: %v", err)
	}

	expectedParent := "system.slice"
	if cgroup.Parent != expectedParent {
		t.Errorf("Expected to have %s as Parent instead of %s", expectedParent, cgroup.Parent)
	}

	expectedScopePrefix := "runc"
	if cgroup.ScopePrefix != expectedScopePrefix {
		t.Errorf("Expected to have %s as ScopePrefix instead of %s", expectedScopePrefix, cgroup.ScopePrefix)
	}

	if cgroup.Name != opts.CgroupName {
		t.Errorf("Expected to have %s as Name instead of %s", opts.CgroupName, cgroup.Name)
	}
}

func TestLinuxCgroupSystemdWithInvalidPath(t *testing.T) {
	cgroupsPath := "/user/cgroups/path/id"

	spec := &specs.Spec{}
	spec.Linux = &specs.Linux{
		CgroupsPath: cgroupsPath,
	}

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: true,
		Spec:             spec,
	}

	_, err := createCgroupConfig(opts)
	if err == nil {
		t.Error("Expected to produce an error if not using the correct format for cgroup paths belonging to systemd")
	}
}
func TestLinuxCgroupsPathSpecified(t *testing.T) {
	cgroupsPath := "/user/cgroups/path/id"

	spec := &specs.Spec{}
	spec.Linux = &specs.Linux{
		CgroupsPath: cgroupsPath,
	}

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: false,
		Spec:             spec,
	}

	cgroup, err := createCgroupConfig(opts)
	if err != nil {
		t.Errorf("Couldn't create Cgroup config: %v", err)
	}

	if cgroup.Path != cgroupsPath {
		t.Errorf("Wrong cgroupsPath, expected '%s' got '%s'", cgroupsPath, cgroup.Path)
	}
}

func TestLinuxCgroupsPathNotSpecified(t *testing.T) {
	spec := &specs.Spec{}
	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: false,
		Spec:             spec,
	}

	cgroup, err := createCgroupConfig(opts)
	if err != nil {
		t.Errorf("Couldn't create Cgroup config: %v", err)
	}

	if cgroup.Path != "" {
		t.Errorf("Wrong cgroupsPath, expected it to be empty string, got '%s'", cgroup.Path)
	}
}

func TestSpecconvExampleValidate(t *testing.T) {
	spec := Example()
	spec.Root.Path = "/"

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: false,
		Spec:             spec,
	}

	config, err := CreateLibcontainerConfig(opts)
	if err != nil {
		t.Errorf("Couldn't create libcontainer config: %v", err)
	}

	validator := validate.New()
	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected specconv to produce valid container config: %v", err)
	}
}

func TestDupNamespaces(t *testing.T) {
	spec := &specs.Spec{
		Root: &specs.Root{
			Path: "rootfs",
		},
		Linux: &specs.Linux{
			Namespaces: []specs.LinuxNamespace{
				{
					Type: "pid",
				},
				{
					Type: "pid",
					Path: "/proc/1/ns/pid",
				},
			},
		},
	}

	_, err := CreateLibcontainerConfig(&CreateOpts{
		Spec: spec,
	})

	if !strings.Contains(err.Error(), "malformed spec file: duplicated ns") {
		t.Errorf("Duplicated namespaces should be forbidden")
	}
}

func TestNonZeroEUIDCompatibleSpecconvValidate(t *testing.T) {
	if _, err := os.Stat("/proc/self/ns/user"); os.IsNotExist(err) {
		t.Skip("userns is unsupported")
	}

	spec := Example()
	spec.Root.Path = "/"
	ToRootless(spec)

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: false,
		Spec:             spec,
		RootlessEUID:     true,
		RootlessCgroups:  true,
	}

	config, err := CreateLibcontainerConfig(opts)
	if err != nil {
		t.Errorf("Couldn't create libcontainer config: %v", err)
	}

	validator := validate.New()
	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected specconv to produce valid rootless container config: %v", err)
	}
}
