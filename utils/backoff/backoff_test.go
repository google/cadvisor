// Copyright 2026 Google Inc. All Rights Reserved.
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

package backoff

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryWithBackoff(t *testing.T) {
	retries := 3
	seenAttempts := map[int]bool{
		0: false,
		1: false,
		2: false,
	}
	expectedAttempts := map[int]bool{
		0: true,
		1: true,
		2: true,
	}

	initialDuration := time.Millisecond
	backoffAmount := 2

	now := time.Now()

	err := RetryWithBackoff(retries, initialDuration, backoffAmount, func(retryAttempt int) (bool, error) {
		seenAttempts[retryAttempt] = true
		return false, nil
	})
	assert.Empty(t, err)

	// Should run through 2 steps, so that's 1 + 2
	expectedDone := 3 * time.Millisecond

	newNow := time.Now()
	assert.Equal(t, seenAttempts, expectedAttempts, "expected all attempts to be retried")
	assert.Equal(t, newNow.After(now.Add(expectedDone)), true, "expected %d to be after %d", newNow, now)
}

func TestRetryWithBackoffStopEarlyWithError(t *testing.T) {
	retries := 3
	seenAttempts := map[int]bool{
		0: false,
		1: false,
		2: false,
		3: false,
	}
	expectedAttempts := map[int]bool{
		0: true,
		1: true,
		2: false,
		3: false,
	}

	initialDuration := time.Millisecond
	backoffAmount := 2

	now := time.Now()

	err := RetryWithBackoff(retries, initialDuration, backoffAmount, func(retryAttempt int) (bool, error) {
		seenAttempts[retryAttempt] = true
		if retryAttempt == 1 {
			return true, fmt.Errorf("failure!")
		}
		return false, nil
	})
	assert.NotEmpty(t, err)

	// Should only run through 1 step, as the second step will mark as failure
	expectedDone := time.Millisecond

	newNow := time.Now()
	assert.Equal(t, seenAttempts, expectedAttempts, "expected all attempts to be retried")
	assert.Equal(t, newNow.After(now.Add(expectedDone)), true, "expected %d to be after %d", newNow, now)
}
