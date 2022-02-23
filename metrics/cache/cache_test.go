package cache_test

import (
	"strings"

	dto "github.com/prometheus/client_model/go"
)

//func TestCachedTGatherer(t *testing.T) {
//	c := cache.NewCachedTGatherer()
//	mfs, done, err := c.Gather()
//	if err != nil {
//		t.Error("gather failed:", err)
//	}
//	done()
//	if got := mfsToString(mfs); got != "" {
//		t.Error("unexpected metric family", got)
//	}
//
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc"}, "help a", prometheus.GaugeValue, 1, nil); err == nil {
//		t.Fatal("required error since StartUpdateSession was not used, got no error")
//	}
//
//	closeSession := c.StartUpdateSession()
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc"}, "help a", prometheus.GaugeValue, 1, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("b", []string{"b", "c"}, []string{"valb", "valc"}, "help b", prometheus.GaugeValue, 2, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc2"}, "help a2", prometheus.GaugeValue, 3, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc3"}, "help a2", prometheus.GaugeValue, 4, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//
//	if err := c.Delete("a", []string{"b", "c"}, []string{"valb", "valc"}); err == nil {
//		t.Fatal("required error since we are in reset mode, got no error")
//	}
//	closeSession()
//
//	mfs, done, err = c.Gather()
//	if err != nil {
//		t.Error("gather failed:", err)
//	}
//	done()
//
//	const expected = "name:\"a\" help:\"help a2\" type:GAUGE " +
//		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:1 > > " +
//		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc2\" > gauge:<value:3 > > " +
//		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc3\" > gauge:<value:4 > > ," +
//		"name:\"b\" help:\"help b\" type:GAUGE " +
//		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:2 > > "
//
//	if got := mfsToString(mfs); got != expected {
//		t.Error("unexpected metric family, got", got)
//	}
//
//	closeSession = c.StartUpdateSession()
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc"}, "help a", prometheus.GaugeValue, 1, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("b", []string{"b", "c"}, []string{"valb", "valc"}, "help b", prometheus.GaugeValue, 2, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc2"}, "help a2", prometheus.GaugeValue, 3, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc3"}, "help a2", prometheus.GaugeValue, 4, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	closeSession()
//
//	mfs, done, err = c.Gather()
//	if err != nil {
//		t.Error("gather failed:", err)
//	}
//	done()
//
//	if got := mfsToString(mfs); got != expected {
//		t.Error("unexpected metric family, got", got)
//	}
//
//	mfs, done, err = c.Gather()
//	if err != nil {
//		t.Error("gather failed:", err)
//	}
//	done()
//
//	if got := mfsToString(mfs); got != expected {
//		t.Error("unexpected metric family, got", got)
//	}
//
//	closeSession = c.StartUpdateSession()
//	if err := c.InsertInPlace("ax", []string{"b", "c"}, []string{"valb", "valc"}, "help ax", prometheus.GaugeValue, 1, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("bx", []string{"b", "c"}, []string{"valb", "valc"}, "help bx", prometheus.GaugeValue, 1, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	closeSession()
//
//	mfs, done, err = c.Gather()
//	if err != nil {
//		t.Error("gather failed:", err)
//	}
//	done()
//
//	if got := mfsToString(mfs); got != "name:\"ax\" help:\"help ax\" type:GAUGE metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" >"+
//		" gauge:<value:1 > > ,name:\"bx\" help:\"help bx\" type:GAUGE metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:1 > > " {
//		t.Error("unexpected metric family, got", got)
//	}
//
//	// Don't insert anything.
//	closeSession = c.StartUpdateSession()
//	closeSession()
//
//	mfs, done, err = c.Gather()
//	if err != nil {
//		t.Error("gather failed:", err)
//	}
//	done()
//	if got := mfsToString(mfs); got != "" {
//		t.Error("unexpected metric family", got)
//	}
//}
//
//func TestWatchCachedTGatherer(t *testing.T) {
//	c := cache.NewWatchCachedTGatherer()
//	mfs, done, err := c.Gather()
//	if err != nil {
//		t.Error("gather failed:", err)
//	}
//	done()
//	if got := mfsToString(mfs); got != "" {
//		t.Error("unexpected metric family", got)
//	}
//
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc"}, "help a", prometheus.GaugeValue, 1, nil); err == nil {
//		t.Fatal("required error since StartUpdateSession was not used, got no error")
//	}
//	if err := c.Delete("a", []string{"b", "c"}, []string{"valb", "valc"}); err == nil {
//		t.Fatal("required error since StartUpdateSession was not used, got no error")
//	}
//
//	closeSession := c.StartUpdateSession()
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc"}, "help a", prometheus.GaugeValue, 1, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("b", []string{"b", "c"}, []string{"valb", "valc"}, "help b", prometheus.GaugeValue, 2, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc2"}, "help a2", prometheus.GaugeValue, 3, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc3"}, "help a2", prometheus.GaugeValue, 4, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.Delete("a", []string{"b", "c"}, []string{"valb", "valc"}); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	closeSession()
//
//	mfs, done, err = c.Gather()
//	if err != nil {
//		t.Error("gather failed:", err)
//	}
//	done()
//
//	const expected = "name:\"a\" help:\"help a2\" type:GAUGE " +
//		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc2\" > gauge:<value:3 > > " +
//		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc3\" > gauge:<value:4 > > ," +
//		"name:\"b\" help:\"help b\" type:GAUGE " +
//		"metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:2 > > "
//
//	if got := mfsToString(mfs); got != expected {
//		t.Error("unexpected metric family, got", got)
//	}
//
//	closeSession = c.StartUpdateSession()
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc"}, "help a", prometheus.GaugeValue, 1, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("b", []string{"b", "c"}, []string{"valb", "valc"}, "help b", prometheus.GaugeValue, 2, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc2"}, "help a2", prometheus.GaugeValue, 3, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.InsertInPlace("a", []string{"b", "c"}, []string{"valb", "valc3"}, "help a2", prometheus.GaugeValue, 4, nil); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.Delete("a", []string{"b", "c"}, []string{"valb", "valc"}); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	closeSession()
//
//	mfs, done, err = c.Gather()
//	if err != nil {
//		t.Error("gather failed:", err)
//	}
//	done()
//
//	if got := mfsToString(mfs); got != expected {
//		t.Error("unexpected metric family, got", got)
//	}
//
//	closeSession = c.StartUpdateSession()
//	if err := c.Delete("a", []string{"b", "c"}, []string{"valb", "valc"}); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.Delete("b", []string{"b", "c"}, []string{"valb", "valc"}); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.Delete("a", []string{"b", "c"}, []string{"valb", "valc2"}); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	if err := c.Delete("a", []string{"b", "c"}, []string{"valb", "valc3"}); err != nil {
//		t.Fatal("unexpected error:", err)
//	}
//	closeSession()
//
//	mfs, done, err = c.Gather()
//	if err != nil {
//		t.Error("gather failed:", err)
//	}
//	done()
//	if got := mfsToString(mfs); got != "" {
//		t.Error("unexpected metric family", got)
//	}
//}

func mfsToString(mfs []*dto.MetricFamily) string {
	ret := make([]string, 0, len(mfs))
	for _, m := range mfs {
		ret = append(ret, m.String())
	}
	return strings.Join(ret, ",")
}
