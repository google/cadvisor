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

package atsd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	atsdStorageDriver "github.com/axibase/atsd-storage-driver/storage"
)

type deduplicationParamsList map[string]atsdStorageDriver.DeduplicationParams

func (self deduplicationParamsList) String() string {
	m := map[string]atsdStorageDriver.DeduplicationParams(self)
	return fmt.Sprint(m)
}

func (self deduplicationParamsList) Set(value string) error {
	groupValues := strings.Split(value, ":")
	if len(groupValues) != 3 {
		return errors.New("Unable to parse a deduplication param value. Expected format: \"group:interval:threshold\"")
	}
	groupName := groupValues[0]

	interval, err := time.ParseDuration(groupValues[1])
	if err != nil {
		return err
	}
	var threshold interface{}
	if strings.HasSuffix(groupValues[2], "%") {
		val, err := strconv.ParseFloat(strings.TrimSuffix(groupValues[2], "%"), 64)
		if err != nil {
			return err
		}
		threshold = atsdStorageDriver.Percent(val)
	} else {
		val, err := strconv.ParseFloat(groupValues[2], 64)
		if err != nil {
			return err
		}
		threshold = atsdStorageDriver.Absolute(val)
	}
	self[groupName] = atsdStorageDriver.DeduplicationParams{Interval: interval, Threshold: threshold}
	return nil
}

type cadvisorParams struct {
	IncludeAllMajorNumbers bool
	UserCgroupsEnabled     bool
	PropertyInterval       time.Duration
	SamplingInterval       time.Duration
	DockerHost             string
}
