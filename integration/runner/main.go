// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/golang/glog"
)

const cadvisorBinary = "cadvisor"

func RunCommand(cmd string, args ...string) error {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("command %q %q failed with error: %v and output: %q", cmd, args, err, output)
	}

	return nil
}

func Run() error {
	start := time.Now()
	defer func() {
		glog.Infof("Execution time %v", time.Since(start))
	}()
	defer glog.Flush()

	host := flag.Args()[0]
	testDir := fmt.Sprintf("/tmp/cadvisor-%d", os.Getpid())
	glog.Infof("Running integration tests in host %q", host)

	// Build cAdvisor.
	glog.Infof("Building cAdvisor...")
	err := RunCommand("godep", "go", "build", "github.com/google/cadvisor")
	if err != nil {
		return err
	}
	defer func() {
		err := RunCommand("rm", cadvisorBinary)
		if err != nil {
			glog.Error(err)
		}
	}()

	// Ship it to the destination host.
	glog.Infof("Pushing cAdvisor binary to remote host...")
	err = RunCommand("gcutil", "ssh", host, "mkdir", "-p", testDir)
	if err != nil {
		return err
	}
	defer func() {
		err := RunCommand("gcutil", "ssh", host, "rm", "-rf", testDir)
		if err != nil {
			glog.Error(err)
		}
	}()
	err = RunCommand("gcutil", "push", host, cadvisorBinary, testDir)
	if err != nil {
		return err
	}

	// TODO(vmarmol): Get logs in case of failures.
	// Start it.
	glog.Infof("Running cAdvisor on the remote host...")
	err = RunCommand("gcutil", "ssh", host, "sudo", "screen", "-d", "-m", path.Join(testDir, cadvisorBinary), "--logtostderr", "&>", "/dev/null")
	if err != nil {
		return err
	}
	defer func() {
		err := RunCommand("gcutil", "ssh", host, "sudo", "killall", cadvisorBinary)
		if err != nil {
			glog.Error(err)
		}
	}()

	// Run the tests.
	glog.Infof("Running integration tests targeting remote host...")
	err = RunCommand("godep", "go", "test", "github.com/google/cadvisor/integration/tests/...", "--host", host)
	if err != nil {
		return err
	}

	glog.Infof("All tests pass!")
	return nil
}

func main() {
	flag.Parse()

	// Check usage.
	if len(flag.Args()) != 1 {
		glog.Fatalf("USAGE: runner <host to test>")
	}

	// Run the tests.
	err := Run()
	if err != nil {
		glog.Fatal(err)
	}
}
