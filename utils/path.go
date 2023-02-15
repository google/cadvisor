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

package utils

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"time"
)

// FileExists returns true if the file exists
func FileExists(file string) bool {
	if _, err := os.Stat(file); err != nil {
		return false
	}
	return true
}

// FilesystemHung returns true if the filesystem appears to be hung.  This is determined by running the `stat` command
// against the path provided and waiting up to the specified timeout for the command to complete.  If it does not
// complete, the command is killed and the function returns false. If the `stat` command is not found on the system,
// the function always returns false.
func FilesystemHung(path string, timeout time.Duration) bool {
	stat, err := exec.LookPath("stat")
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// we don't actually care if the command suceeds or fails, just if it hangs
	_ = exec.CommandContext(ctx, stat, path).Run()
	return errors.Is(ctx.Err(), context.DeadlineExceeded)
}
