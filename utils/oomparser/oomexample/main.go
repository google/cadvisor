// Copyright 2014 Google Inc. All Rights Reserved.
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

//go:build linux

package main

import (
	"flag"

	"k8s.io/klog/v2"

	"github.com/google/cadvisor/utils/oomparser"
)

// demonstrates how to run oomparser.OomParser to get OomInstance information
func main() {
	klog.InitFlags(nil)
	// Opt into the new klog behavior so that -stderrthreshold is honored even
	// when -logtostderr=true (the default).
	// Ref: kubernetes/klog#212, kubernetes/klog#432
	flag.Set("legacy_stderr_threshold_behavior", "false") //nolint:errcheck
	flag.Set("stderrthreshold", "INFO")                   //nolint:errcheck
	flag.Parse()
	// out is a user-provided channel from which the user can read incoming
	// OomInstance objects
	outStream := make(chan *oomparser.OomInstance)
	oomLog, err := oomparser.New()
	if err != nil {
		klog.Infof("Couldn't make a new oomparser. %v", err)
	} else {
		go oomLog.StreamOoms(outStream)
		// demonstration of how to get oomLog's list of oomInstances or access
		// the user-declared oomInstance channel, here called outStream
		for oomInstance := range outStream {
			klog.Infof("Reading the buffer. Output is %v", oomInstance)
		}
	}
}
