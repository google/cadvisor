package storage

import (
	"github.com/axibase/atsd-api-go/net"
	"math"
	"sort"
	"time"
)

type sample struct {
	Time  net.Millis
	Value net.Number
}

type Percent float64
type Absolute float64

type DeduplicationParams struct {
	Threshold interface{}
	Interval  time.Duration
}
type DataCompacter struct {
	Buffer      map[string]map[string]sample
	GroupParams map[string]DeduplicationParams
}

func (self *DataCompacter) Filter(group string, seriesCommands []*net.SeriesCommand) []*net.SeriesCommand {
	output := []*net.SeriesCommand{}

	if _, ok := self.Buffer[group]; ok {
		for _, seriesCommand := range seriesCommands {
			if seriesCommand.Timestamp() != nil {
				timestamp := *seriesCommand.Timestamp()
				var newSc *net.SeriesCommand
				for metric, val := range seriesCommand.Metrics() {
					key := getKey(seriesCommand.Entity(), metric, seriesCommand.Tags())
					if _, ok := self.Buffer[group][key]; !ok ||
						hasChangedEnough(self.Buffer[group][key].Value, val, self.GroupParams[group].Threshold) ||
						time.Duration(timestamp-self.Buffer[group][key].Time)*time.Millisecond >= self.GroupParams[group].Interval ||
						time.Duration(timestamp-self.Buffer[group][key].Time)*time.Millisecond < 0 {

						if newSc == nil {
							newSc = net.NewSeriesCommand(seriesCommand.Entity(), metric, val).SetTimestamp(timestamp)
							for name, val := range seriesCommand.Tags() {
								newSc.SetTag(name, val)
							}
						} else {
							newSc.SetMetricValue(metric, val)
						}

						if _, ok := self.Buffer[group][key]; !ok ||
							time.Duration(timestamp-self.Buffer[group][key].Time)*time.Millisecond > 0 {

							self.Buffer[group][key] = sample{Time: timestamp, Value: val}
						}
					}
				}
				if newSc != nil {
					output = append(output, newSc)
				}
			} else {
				output = append(output, seriesCommand)
			}
		}
	} else {
		output = append(output, seriesCommands...)
	}
	return output
}

func hasChangedEnough(oldValue, newValue net.Number, threshold interface{}) bool {
	switch thrVal := threshold.(type) {
	case Percent:
		switch val1 := oldValue.(type) {
		case net.Float32:
			return math.Abs(val1.Float64()-newValue.Float64())/oldValue.Float64() > float64(thrVal)
		case net.Float64:
			return math.Abs(val1.Float64()-newValue.Float64())/oldValue.Float64() > float64(thrVal)
		case net.Int32:
			return math.Abs(float64(val1.Int64()-val1.Int64()))/oldValue.Float64() > float64(thrVal)
		case net.Int64:
			return math.Abs(float64(val1.Int64()-val1.Int64()))/oldValue.Float64() > float64(thrVal)
		default:
			return math.Abs(val1.Float64()-newValue.Float64())/oldValue.Float64() > float64(thrVal)
		}
	case Absolute:
		switch val1 := oldValue.(type) {
		case net.Float32:
			return math.Abs(val1.Float64()-newValue.Float64()) > float64(thrVal)
		case net.Float64:
			return math.Abs(val1.Float64()-newValue.Float64()) > float64(thrVal)
		case net.Int32:
			return math.Abs(float64(val1.Int64()-val1.Int64())) > float64(thrVal)
		case net.Int64:
			return math.Abs(float64(val1.Int64()-val1.Int64())) > float64(thrVal)
		default:
			return math.Abs(val1.Float64()-newValue.Float64()) > float64(thrVal)
		}
	default:
		panic("Undefined threshold value")
	}

}

func getKey(entity, metric string, tags map[string]string) string {
	key := entity + metric
	tagsList := []string{}
	for tagName, tagValue := range tags {
		tagsList = append(tagsList, tagName+"="+tagValue)
	}
	sort.Strings(tagsList)
	for i := range tagsList {
		key += tagsList[i]
	}
	return key
}
