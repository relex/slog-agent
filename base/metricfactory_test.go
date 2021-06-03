package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricFactory(t *testing.T) {
	mfactory := NewMetricFactory("testmetricfactory_", []string{"test"}, []string{"TestMetricFactory"})
	mfactory.AddOrGetCounter("mycounter", "Help mycounter", []string{"name"}, []string{"foo"}).Add(3)
	mfactory.AddOrGetCounter("mycounter", "Help mycounter", []string{"name"}, []string{"foo"}).Add(4)
	mfactory.AddOrGetCounterVec("mycountervec", "Help mycountervec", []string{"category"}, nil).WithLabelValues("book").Add(5)
	subfactory := mfactory.NewSubFactory("child1_", []string{"type"}, []string{"goroutine"})
	subfactory.AddOrGetGauge("childgauge", "Help childgauge", []string{"name"}, []string{"bar"}).Add(13)
	subfactory.AddOrGetGaugeVec("childgaugevec", "Help childgaugevec", []string{"class"}, nil).WithLabelValues("X").Add(14)
	subfactory.AddOrGetGaugeVec("childgaugevec", "Help childgaugevec", []string{"class"}, nil).WithLabelValues("X").Add(1)
	subfactory.AddOrGetGaugeVec("childgaugevec", "Help childgaugevec", []string{"class"}, nil).WithLabelValues("Y").Add(16)
	metrics, merr := mfactory.DumpMetrics(true)
	assert.Nil(t, merr)
	assert.Equal(t, `testmetricfactory_child1_childgauge{name="bar",test="TestMetricFactory",type="goroutine"} 13
testmetricfactory_child1_childgaugevec{class="X",test="TestMetricFactory",type="goroutine"} 15
testmetricfactory_child1_childgaugevec{class="Y",test="TestMetricFactory",type="goroutine"} 16
testmetricfactory_mycounter{name="foo",test="TestMetricFactory"} 7
testmetricfactory_mycountervec{category="book",test="TestMetricFactory"} 5
`, metrics)
}
