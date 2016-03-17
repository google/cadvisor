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
	"github.com/axibase/atsd-api-go/net"
)

type SeriesType string

const (
	History           SeriesType = "HISTORY"
	Forecast          SeriesType = "FORECAST"
	ForecastDeviation SeriesType = "FORECAST_DEVIATION"
)

type GroupType string

const (
	StatCount             GroupType = "COUNT"
	StatMin               GroupType = "MIN"
	StatMax               GroupType = "MAX"
	StatAvg               GroupType = "AVG"
	StatSum               GroupType = "SUM"
	StatPercentile999     GroupType = "PERCENTILE_999"
	StatPercentile995     GroupType = "PERCENTILE_995"
	StatPercentile99      GroupType = "PERCENTILE_99"
	StatPercentile95      GroupType = "PERCENTILE_95"
	StatPercentile90      GroupType = "PERCENTILE_90"
	StatPercentile75      GroupType = "PERCENTILE_75"
	StatPercentile50      GroupType = "PERCENTILE_50"
	StatMedian            GroupType = "MEDIAN"
	StatStandardDeviation GroupType = "STANDARD_DEVIATION"
	StatMinValueTime      GroupType = "MIN_VALUE_TIME"
	StatMaxValueTime      GroupType = "MAX_VALUE_TIME"
)

type InterpolationType string

const (
	None   InterpolationType = "NONE"
	Step   InterpolationType = "STEP"
	Linear InterpolationType = "LINEAR"
)

type Period struct {
	Count uint `json:"count"`
	Unit  Unit `json:"unit"`
}

type Group struct {
	Type        GroupType         `json:"type"`
	Interpolate InterpolationType `json:"interpolate,omitempty"`
	Truncate    bool              `json:"truncate,omitempty"`
	Period      *Period           `json:"period,omitempty"`
	Order       uint              `json:"order,omitempty"`
}

type Rate struct {
	Period  *Period `json:"period,omitempty"`
	Counter bool    `json:"counter"`
}

type AggregationType string

const (
	AgDetail            AggregationType = "DETAIL"
	AgCount             AggregationType = "COUNT"
	AgMin               AggregationType = "MIN"
	AgMax               AggregationType = "MAX"
	AgAvg               AggregationType = "AVG"
	AgSum               AggregationType = "SUM"
	AgPercentile999     AggregationType = "PERCENTILE_999"
	AgPercentile995     AggregationType = "PERCENTILE_995"
	AgPercentile99      AggregationType = "PERCENTILE_99"
	AgPercentile95      AggregationType = "PERCENTILE_95"
	AgPercentile90      AggregationType = "PERCENTILE_90"
	AgPercentile75      AggregationType = "PERCENTILE_75"
	AgPercentile50      AggregationType = "PERCENTILE_50"
	AgMedian            AggregationType = "MEDIAN"
	AgStandardDeviation AggregationType = "STANDARD_DEVIATION"
	AgFirst             AggregationType = "FIRST"
	AgLast              AggregationType = "LAST"
	AgDelta             AggregationType = "DELTA"
	AgWavg              AggregationType = "WAVG"
	AgWtavg             AggregationType = "WTAVG"
	AgThresholdCount    AggregationType = "THRESHOLD_COUNT"
	AgThresholdDuration AggregationType = "THRESHOLD_DURATION"
	AgThreshold_Percent AggregationType = "THRESHOLD_PERCENT"
	AgMinValueTime      AggregationType = "MIN_VALUE_TIME"
	AgMaxValueTime      AggregationType = "MAX_VALUE_TIME"
)

type Unit string

const (
	Millisecond Unit = "MILLISECOND"
	Second      Unit = "SECOND"
	Minute      Unit = "MINUTE"
	Hour        Unit = "HOUR"
	Day         Unit = "DAY"
	Week        Unit = "WEEK"
	Month       Unit = "MONTH"
	Quarter     Unit = "QUARTER"
	Year        Unit = "YEAR"
)

type Threshold struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

type Calendar struct {
	Name string `json:"name"`
}

type WorkingMinutes struct {
	Start uint `json:"start"`
	End   uint `json:"end"`
}

type Aggregation struct {
	Types          []AggregationType `json:"types,omitempty"`
	Type           AggregationType   `json:"type,omitempty"`
	Period         Period            `json:"period,omitempty"`
	Interpolate    InterpolationType `json:"interpolate,omitempty"`
	Threshold      *Threshold        `json:"threshold,omitempty"`
	Calendar       *Calendar         `json:"calendar,omitempty"`
	WorkingMinutes *WorkingMinutes   `json:"workingMinutes,omitempty"`
	Counter        bool              `json:"counter,omitempty"`
	Order          uint              `json:"order,omitempty"`
}

type SeriesQuery struct {
	StartTime    net.Millis          `json:"startTime,omitempty"`
	EndTime      net.Millis          `json:"endTime,omitempty"`
	StartDate    string              `json:"startDate,omitempty"`
	EndDate      string              `json:"endDate,omitempty"`
	Interval     string              `json:"interval,omitempty"`
	Limit        uint64              `json:"limit,omitempty"`
	Entity       string              `json:"entity"`
	Metric       string              `json:"metric"`
	Last         bool                `json:"last,omitempty"`
	Cache        bool                `json:"cache,omitempty"`
	Type         SeriesType          `json:"type,omitempty"`
	ForecastName string              `json:"forecastName,omitempty"`
	Group        *Group              `json:"group,omitempty"`
	Rate         *Rate               `json:"rate,omitempty"`
	Aggregate    *Aggregation        `json:"aggregate,omitempty"`
	RequestId    *string             `json:"requestId,omitempty"`
	Tags         map[string][]string `json:"tags,omitempty"`
}
