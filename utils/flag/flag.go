// Copyright 2016 Google Inc. All Rights Reserved.
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

package flag

import (
	"flag"
	"fmt"
	"strconv"
	"time"
)

// Registered cAdvisor flags.
var Flags = flag.NewFlagSet("cAdvisor", flag.ExitOnError)

// Bool adds a cAdvisor-scoped bool flag.
func Bool(name string, value bool, usage string) *bool {
	return Flags.Bool(name, value, usage)
}

// Duration adds a cAdvisor-scoped duration flag.
func Duration(name string, value time.Duration, usage string) *time.Duration {
	return Flags.Duration(name, value, usage)
}

// Int adds a cAdvisor-scoped int flag.
func Int(name string, value int, usage string) *int {
	return Flags.Int(name, value, usage)
}

// String adds a cAdvisor-scoped string flag.
func String(name string, value string, usage string) *string {
	return Flags.String(name, value, usage)
}

// SetBool sets the value of the named flag. If the flag ultimately parsed, value will be the new
// default. If the flag is never parsed (e.g. not added to the CommandLine set), then the parameter
// will still be set to value. Must be called before any Add or Parse methods.
func SetBool(name string, value bool) error {
	return Flags.Set(name, strconv.FormatBool(value))
}

// SetDuration sets the value of the named flag. If the flag ultimately parsed, value will be the new
// default. If the flag is never parsed (e.g. not added to the CommandLine set), then the parameter
// will still be set to value. Must be called before any Add or Parse methods.
func SetDuration(name string, value time.Duration) error {
	return Flags.Set(name, value.String())
}

// SetInt sets the value of the named flag. If the flag ultimately parsed, value will be the new
// default. If the flag is never parsed (e.g. not added to the CommandLine set), then the parameter
// will still be set to value. Must be called before any Add or Parse methods.
func SetInt(name string, value int) error {
	return Flags.Set(name, strconv.Itoa(value))
}

// SetString sets the value of the named flag. If the flag ultimately parsed, value will be the new
// default. If the flag is never parsed (e.g. not added to the CommandLine set), then the parameter
// will still be set to value. Must be called before any Add or Parse methods.
func SetString(name string, value string) error {
	return Flags.Set(name, value)
}

// Adds the specified flags to the given FlagSet. Returns an error if any flags are missing.
func AddFlags(fs *flag.FlagSet, flags []string) error {
	visited := make(map[string]bool, len(flags))
	for _, f := range flags {
		visited[f] = false
	}
	Flags.VisitAll(func(f *flag.Flag) {
		if _, ok := visited[f.Name]; ok {
			addFlag(fs, f)
			visited[f.Name] = true
		}
	})
	missing := []string{}
	for name, found := range visited {
		if !found {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing flags: %v", missing)
	}
	return nil
}

// Adds all registered flags to the given FlagSet. Calling more than once on the
// same flag set will result in a panic.
// NOTE: Flags added after AddAllFlags is called will not be added to the
// FlagSet. This means that if AddAllFlags is called in a package init()
// function, there is an implicit dependency on the import ordering.
func AddAllFlags(fs *flag.FlagSet) {
	Flags.VisitAll(func(f *flag.Flag) {
		addFlag(fs, f)
	})
}

func addFlag(fs *flag.FlagSet, f *flag.Flag) {
	fs.Var(f.Value, f.Name, f.Usage)
}
