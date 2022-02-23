package cache_test

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cadvisor/metrics/cache"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var testEntries = []*cache.Metric{
	{FQName: proto.String("a"), LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}, Help: proto.String("help a"), ValueType: prometheus.GaugeValue, Value: 1},
	{FQName: proto.String("b"), LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}, Help: proto.String("help b"), ValueType: prometheus.GaugeValue, Value: 2},
	{FQName: proto.String("a"), LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc2"}, Help: proto.String("help a2"), ValueType: prometheus.GaugeValue, Value: 3},
	{FQName: proto.String("a"), LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc3"}, Help: proto.String("help a3"), ValueType: prometheus.GaugeValue, Value: 4},
}

func TestCachedTGatherer(t *testing.T) {
	c := cache.NewCachedTGatherer()
	mfs, done, err := c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()
	if got := mfsToString(mfs); got != "" {
		t.Error("unexpected metric family", got)
	}

	if err := c.InsertInPlace(testEntries[0]); err == nil {
		t.Fatal("required error since StartUpdateSession was not used, got no error")
	}

	closeSession := c.StartUpdateSession()
	if err := c.InsertInPlace(testEntries[0]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[1]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[2]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[3]); err != nil {
		t.Fatal("unexpected error:", err)
	}

	if err := c.Delete(testEntries[0]); err == nil {
		t.Fatal("required error since we are in reset mode, got no error")
	}
	closeSession()

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()

	const expected = "name:\"a\" help:\"help a2\" type:GAUGE " +
		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:1 > > " +
		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc2\" > gauge:<value:3 > > " +
		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc3\" > gauge:<value:4 > > ," +
		"name:\"b\" help:\"help b\" type:GAUGE " +
		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:2 > > "

	if got := mfsToString(mfs); got != expected {
		t.Error("unexpected metric family, got", got)
	}

	closeSession = c.StartUpdateSession()
	if err := c.InsertInPlace(testEntries[0]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[1]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[2]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[3]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	closeSession()

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()

	if got := mfsToString(mfs); got != expected {
		t.Error("unexpected metric family, got", got)
	}

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()

	if got := mfsToString(mfs); got != expected {
		t.Error("unexpected metric family, got", got)
	}

	closeSession = c.StartUpdateSession()
	if err := c.InsertInPlace(&cache.Metric{
		FQName: "ax", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}, Help: "help ax", ValueType: prometheus.GaugeValue, Value: 1,
	}); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(&cache.Metric{
		FQName: "bx", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}, Help: "help bx", ValueType: prometheus.GaugeValue, Value: 1,
	}); err != nil {
		t.Fatal("unexpected error:", err)
	}
	closeSession()

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()

	if got := mfsToString(mfs); got != "name:\"ax\" help:\"help ax\" type:GAUGE metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" >"+
		" gauge:<value:1 > > ,name:\"bx\" help:\"help bx\" type:GAUGE metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:1 > > " {
		t.Error("unexpected metric family, got", got)
	}

	// Don't insert anything.
	closeSession = c.StartUpdateSession()
	closeSession()

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()
	if got := mfsToString(mfs); got != "" {
		t.Error("unexpected metric family", got)
	}
}

func TestWatchCachedTGatherer(t *testing.T) {
	c := cache.NewWatchCachedTGatherer()
	mfs, done, err := c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()
	if got := mfsToString(mfs); got != "" {
		t.Error("unexpected metric family", got)
	}

	if err := c.InsertInPlace(testEntries[0]); err == nil {
		t.Fatal("required error since StartUpdateSession was not used, got no error")
	}
	if err := c.Delete(testEntries[0]); err == nil {
		t.Fatal("required error since StartUpdateSession was not used, got no error")
	}

	closeSession := c.StartUpdateSession()
	if err := c.InsertInPlace(testEntries[0]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[1]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[2]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[3]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.Delete(testEntries[0]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	closeSession()

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()

	const expected = "name:\"a\" help:\"help a2\" type:GAUGE " +
		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc2\" > gauge:<value:3 > > " +
		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc3\" > gauge:<value:4 > > ," +
		"name:\"b\" help:\"help b\" type:GAUGE " +
		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:2 > > "

	if got := mfsToString(mfs); got != expected {
		t.Error("unexpected metric family, got", got)
	}

	closeSession = c.StartUpdateSession()
	if err := c.InsertInPlace(testEntries[0]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[1]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[2]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.InsertInPlace(testEntries[3]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.Delete(testEntries[0]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	closeSession()

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()

	if got := mfsToString(mfs); got != expected {
		t.Error("unexpected metric family, got", got)
	}

	closeSession = c.StartUpdateSession()
	if err := c.Delete(testEntries[0]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.Delete(testEntries[1]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.Delete(testEntries[2]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if err := c.Delete(testEntries[3]); err != nil {
		t.Fatal("unexpected error:", err)
	}
	closeSession()

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()
	if got := mfsToString(mfs); got != "" {
		t.Error("unexpected metric family", got)
	}
}

func mfsToString(mfs []*dto.MetricFamily) string {
	ret := make([]string, 0, len(mfs))
	for _, m := range mfs {
		ret = append(ret, m.String())
	}
	return strings.Join(ret, ",")
}
