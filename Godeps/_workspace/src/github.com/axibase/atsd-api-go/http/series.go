/*
* Copyright 2015 Axibase Corporation or its affiliates. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License").
* You may not use this file except in compliance with the License.
* A copy of the License is located at
*
* https://www.axibase.com/atsd/axibase-apache-2.0.pdf
*
* or in the "license" file accompanying this file. This file is distributed
* on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
* express or implied. See the License for the specific language governing
* permissions and limitations under the License.
 */

package http

import (
	"bytes"
	"encoding/json"
	"github.com/axibase/atsd-api-go/net"
	"strings"
	"time"
)

type Sample struct {
	T net.Millis `json:"t"`
	V net.Number `json:"v"`
}

func (self *Sample) UnmarshalJSON(data []byte) error {
	var jsonMap map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&jsonMap); err != nil {
		return err
	}
	self.T, _ = jsonMap["t"].(net.Millis)
	switch value := jsonMap["v"].(type) {
	case json.Number:
		strRep := value.String()
		if strings.Contains(strRep, ".") {
			temp, _ := value.Float64()
			self.V = net.Float64(temp)
		} else {
			temp, _ := value.Int64()
			self.V = net.Int64(temp)
		}
	default:
		panic(value)
	}
	return nil
}

type ForecastMeta struct {
	Timestamp         net.Millis    `json:"timestamp"`
	AveragingInterval time.Duration `json:"averagingInterval"`
	Alpha             float64       `json:"alpha"`
	Beta              float64       `json:"beta"`
	Gamma             float64       `json:"gamma"`
	Period            string        `json:"period"`
	stdDev            float64       `json:"stdDev"`
}

type Series struct {
	Entity  string            `json:"entity"`
	Metric  string            `json:"metric"`
	Tags    map[string]string `json:"tags,omitempty"`
	Warning string            `json:"warning,omitempty"`
	Data    []*Sample         `json:"data"`

	Type         SeriesType    `json:"type"`
	ForecastName string        `json:"forecastName,omitempty"`
	RequestId    string        `json:"requestId,omitempty"`
	Meta         *ForecastMeta `json:"meta,omitempty"`
	Aggregate    *Aggregation  `json:"aggregate,omitempty"`
}
