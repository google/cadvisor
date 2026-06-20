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

package v1

import "github.com/google/cadvisor/lib/model"

// Type of metric being exported.
type MetricType = model.MetricType

const (
	// Instantaneous value. May increase or decrease.
	MetricGauge MetricType = "gauge"

	// A counter-like value that is only expected to increase.
	MetricCumulative MetricType = "cumulative"
)

// DataType for metric being exported.
type DataType = model.DataType

const (
	IntType   DataType = "int"
	FloatType DataType = "float"
)

// Spec for custom metric.
type MetricSpec = model.MetricSpec

// An exported metric.
type MetricValBasic = model.MetricValBasic

// An exported metric.
type MetricVal = model.MetricVal
