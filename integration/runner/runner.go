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
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/cadvisor/integration/common"

	"github.com/golang/glog"
)

const cadvisorBinary = "cadvisor"

var cadvisorTimeout = flag.Duration("cadvisor_timeout", 15*time.Second, "Time to wait for cAdvisor to come up on the remote host")
var port = flag.Int("port", 8080, "Port in which to start cAdvisor in the remote host")
var testRetryCount = flag.Int("test-retry-count", 3, "Number of times to retry failed tests before failing.")
var testRetryWhitelist = flag.String("test-retry-whitelist", "", "Path to newline separated list of regexexp for test failures that should be retried.  If empty, no tests are retried.")
var retryRegex *regexp.Regexp

func RunCommand(cmd string, args ...string) error {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("command %q %q failed with error: %v and output: %q", cmd, args, err, output)
	}

	return nil
}

func PushAndRunTests(host, testDir string) error {
	// Push binary.
	glog.Infof("Pushing cAdvisor binary to %q...", host)
	args := common.GetGCComputeArgs("ssh", host, "--", "mkdir", "-p", testDir)
	err := RunCommand("gcloud", args...)
	if err != nil {
		return fmt.Errorf("failed to make remote testing directory: %v", err)
	}
	defer func() {
		args := common.GetGCComputeArgs("ssh", host, "--", "rm", "-rf", testDir)
		err := RunCommand("gcloud", args...)
		if err != nil {
			glog.Errorf("Failed to cleanup test directory: %v", err)
		}
	}()
	args = common.GetGCComputeArgs("copy-files", cadvisorBinary, fmt.Sprintf("%s:%s", host, testDir))
	err = RunCommand("gcloud", args...)
	if err != nil {
		return fmt.Errorf("failed to copy binary: %v", err)
	}

	// TODO(vmarmol): Get logs in case of failures.
	// Start cAdvisor.
	glog.Infof("Running cAdvisor on %q...", host)
	portStr := strconv.Itoa(*port)
	errChan := make(chan error)
	go func() {
		args = common.GetGCComputeArgs("ssh", host, "--", fmt.Sprintf("sudo %s --port %s --logtostderr", path.Join(testDir, cadvisorBinary), portStr))
		err = RunCommand("gcloud", args...)
		if err != nil {
			errChan <- fmt.Errorf("error running cAdvisor: %v", err)
		}
	}()
	defer func() {
		args = common.GetGCComputeArgs("ssh", host, "--", "sudo", "pkill", cadvisorBinary)
		err := RunCommand("gcloud", args...)
		if err != nil {
			glog.Errorf("Failed to cleanup: %v", err)
		}
	}()

	ipAddress, err := common.GetGceIp(host)
	if err != nil {
		return fmt.Errorf("failed to get GCE IP: %v", err)
	}

	// Wait for cAdvisor to come up.
	endTime := time.Now().Add(*cadvisorTimeout)
	done := false
	for endTime.After(time.Now()) && !done {
		select {
		case err := <-errChan:
			// Quit early if there was an error.
			return err
		case <-time.After(500 * time.Millisecond):
			// Stop waiting when cAdvisor is healthy..
			resp, err := http.Get(fmt.Sprintf("http://%s:%s/healthz", ipAddress, portStr))
			if err == nil && resp.StatusCode == http.StatusOK {
				done = true
				break
			}
		}
	}
	if !done {
		return fmt.Errorf("timed out waiting for cAdvisor to come up at host %q", host)
	}

	// Run the tests in a retry loop.
	glog.Infof("Running integration tests targeting %q...", host)
	for i := 0; i <= *testRetryCount; i++ {
		// Check if this is a retry
		if i > 0 {
			time.Sleep(time.Second * 15) // Wait 15 seconds before retrying
			glog.Warningf("Retrying (%d of %d) tests on host %s due to error %v", i, *testRetryCount, host, err)
		}
		// Run the command
		err = RunCommand("godep", "go", "test", "github.com/google/cadvisor/integration/tests/...", "--host", host, "--port", portStr)
		if err == nil {
			// On success, break out of retry loop
			break
		}

		// Only retry on test failures caused by these known flaky failure conditions
		if retryRegex == nil || !retryRegex.Match([]byte(err.Error())) {
			glog.Warningf("Skipping retry for tests on host %s because error is not whitelisted: %s", host, err.Error())
			break
		}
	}
	if err != nil {
		err = fmt.Errorf("error on host %s: %v", host, err)
	}

	return err
}

func Run() error {
	start := time.Now()
	defer func() {
		glog.Infof("Execution time %v", time.Since(start))
	}()
	defer glog.Flush()

	hosts := flag.Args()
	testDir := fmt.Sprintf("/tmp/cadvisor-%d", os.Getpid())
	glog.Infof("Running integration tests on host(s) %q", strings.Join(hosts, ","))

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

	// Run test on all hosts in parallel.
	var wg sync.WaitGroup
	allErrors := make([]error, 0)
	var allErrorsLock sync.Mutex
	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			err := PushAndRunTests(host, testDir)
			if err != nil {
				func() {
					allErrorsLock.Lock()
					defer allErrorsLock.Unlock()
					allErrors = append(allErrors, err)
				}()
			}
		}(host)
	}
	wg.Wait()

	if len(allErrors) != 0 {
		var buffer bytes.Buffer
		for i, err := range allErrors {
			buffer.WriteString(fmt.Sprintf("Error %d: ", i))
			buffer.WriteString(err.Error())
			buffer.WriteString("\n")
		}
		return errors.New(buffer.String())
	}

	glog.Infof("All tests pass!")
	return nil
}

// initRetryWhitelist initializes the whitelist of test failures that can be retried.
func initRetryWhitelist() {
	if *testRetryWhitelist == "" {
		return
	}

	file, err := os.Open(*testRetryWhitelist)
	if err != nil {
		glog.Fatal(err)
	}
	defer file.Close()

	retryStrings := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if text != "" {
			retryStrings = append(retryStrings, text)
		}
	}
	if err := scanner.Err(); err != nil {
		glog.Fatal(err)
	}
	retryRegex = regexp.MustCompile(strings.Join(retryStrings, "|"))
}

func main() {
	flag.Parse()

	// Check usage.
	if len(flag.Args()) == 0 {
		glog.Fatalf("USAGE: runner <hosts to test>")
	}
	initRetryWhitelist()

	// Run the tests.
	err := Run()
	if err != nil {
		glog.Fatal(err)
	}
}
