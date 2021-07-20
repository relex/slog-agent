package base

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter"
)

// MetricFactory manages Prometheus metrics
type MetricFactory struct {
	namePrefix        string
	parentLabelNames  []string
	parentLabelValues []string
	registryLock      *sync.Mutex
	registry          map[string]prometheus.Collector
}

// NewMetricFactory creates a factory with prefix for metrics names and fixed labels for all metrics created from this new factory
func NewMetricFactory(prefix string, labelNames []string, labelValues []string) *MetricFactory {
	if len(labelNames) != len(labelValues) {
		logger.Panicf("different len of labelNames (%s) and labelValues (%s)",
			strings.Join(labelNames, ","), strings.Join(labelValues, ","))
	}
	return &MetricFactory{
		namePrefix:        prefix,
		parentLabelNames:  labelNames,
		parentLabelValues: labelValues,
		registryLock:      &sync.Mutex{},
		registry:          make(map[string]prometheus.Collector, 1000),
	}
}

// NewSubFactory creates a sub-factory which inherits the parent's prefix and fixed labels,
// with more prefix and fixed labels added to all metrics created from this new sub-factory
func (factory *MetricFactory) NewSubFactory(prefix string, labelNames []string, labelValues []string) *MetricFactory {
	fullPrefix, allLabelNames, allLabelValues := factory.concatNameAndLabels(prefix, labelNames, labelValues)
	return &MetricFactory{
		namePrefix:        fullPrefix,
		parentLabelNames:  allLabelNames,
		parentLabelValues: allLabelValues,
		registryLock:      factory.registryLock,
		registry:          factory.registry,
	}
}

// AddOrGetCounter adds or gets a counter
func (factory *MetricFactory) AddOrGetCounter(name string, help string, labelNames []string, labelValues []string) promexporter.RWCounter {
	if len(labelNames) != len(labelValues) {
		logger.Panicf("different lengths of labelNames (%s) and labelValues (%s)",
			strings.Join(labelNames, ","), strings.Join(labelValues, ","))
	}
	return factory.AddOrGetCounterVec(name, help, labelNames, labelValues).WithLabelValues()
}

// AddOrGetCounterVec adds or gets a counter-vec with leftmost label values
func (factory *MetricFactory) AddOrGetCounterVec(name string, help string, labelNames []string, leftmostLabelValues []string) *promexporter.RWCounterVec {
	fullName, allLabelNames, allLeftmostLabelValues := factory.concatNameAndLabels(name, labelNames, leftmostLabelValues)

	factory.registryLock.Lock()
	var counterVec *promexporter.RWCounterVec
	if metricVec, ok := factory.registry[fullName]; ok {
		counterVec = metricVec.(*promexporter.RWCounterVec)
	} else {
		counterOps := prometheus.CounterOpts{}
		counterOps.Name = fullName
		counterOps.Help = help
		counterVec = promexporter.NewRWCounterVec(counterOps, allLabelNames)
		factory.registry[fullName] = (prometheus.Collector)(counterVec)
		if err := prometheus.Register(counterVec); err != nil {
			logger.Panicf("failed to register counter-vec '%s': %s", fullName, err.Error())
		}
	}
	factory.registryLock.Unlock()

	curryLabels := buildLabels(allLabelNames, allLeftmostLabelValues)
	curriedCounterVec, cerr := counterVec.CurryWith(curryLabels)
	if cerr != nil {
		logger.Panicf("failed to curry counter-vec '%s' with %s: %s", fullName, curryLabels, cerr.Error())
	}
	return curriedCounterVec
}

// AddOrGetGauge adds or gets a gauge
//
// Gauges must be updated by Add/Sub not Set, because there could be multiple updaters
func (factory *MetricFactory) AddOrGetGauge(name string, help string, labelNames []string, labelValues []string) promexporter.RWGauge {
	if len(labelNames) != len(labelValues) {
		logger.Panicf("different lengths of labelNames (%s) and labelValues (%s)",
			strings.Join(labelNames, ","), strings.Join(labelValues, ","))
	}
	return factory.AddOrGetGaugeVec(name, help, labelNames, labelValues).WithLabelValues()
}

// AddOrGetGaugeVec adds or gets a gauge-vec with leftmost label values
//
// Gauges must be updated by Add/Sub not Set, because there could be multiple updaters
func (factory *MetricFactory) AddOrGetGaugeVec(name string, help string, labelNames []string, leftmostLabelValues []string) *promexporter.RWGaugeVec {
	fullName, allLabelNames, allLeftmostLabelValues := factory.concatNameAndLabels(name, labelNames, leftmostLabelValues)

	factory.registryLock.Lock()
	var gaugeVec *promexporter.RWGaugeVec
	if metricVec, ok := factory.registry[fullName]; ok {
		gaugeVec = metricVec.(*promexporter.RWGaugeVec)
	} else {
		gaugeOpts := prometheus.GaugeOpts{}
		gaugeOpts.Name = fullName
		gaugeOpts.Help = help
		gaugeVec = promexporter.NewRWGaugeVec(gaugeOpts, allLabelNames)
		factory.registry[fullName] = (prometheus.Collector)(gaugeVec)
		if err := prometheus.Register(gaugeVec); err != nil {
			logger.Panicf("failed to register gauge-vec '%s': %s", fullName, err.Error())
		}
	}
	factory.registryLock.Unlock()

	curryLabels := buildLabels(allLabelNames, allLeftmostLabelValues)
	curriedGaugeVec, cerr := gaugeVec.CurryWith(curryLabels)
	if cerr != nil {
		logger.Panicf("failed to curry gauge-vec '%s' with %s: %s", fullName, curryLabels, cerr.Error())
	}
	return curriedGaugeVec
}

// DumpMetrics dumps all metrics created in this factory and derived sub-factories into the .prom text format without comments
//
// For testing only
func (factory *MetricFactory) DumpMetrics(includeZeroValues bool) (string, error) {
	gatherer, err := func() (*prometheus.Registry, error) {
		g := prometheus.NewPedanticRegistry()
		factory.registryLock.Lock()
		defer factory.registryLock.Unlock()
		for name, vec := range factory.registry {
			if !strings.HasPrefix(name, factory.namePrefix) {
				continue
			}
			if err := g.Register(vec); err != nil {
				return nil, fmt.Errorf("failed to add metric '%s' to gatherer: %w", name, err)
			}
		}
		return g, nil
	}()
	if err != nil {
		return "", nil
	}
	metricFamilies, err := gatherer.Gather()
	if err != nil {
		return "", fmt.Errorf("failed to gather metrics: %w", err)
	}
	writer := &bytes.Buffer{}
	for _, mf := range metricFamilies {
		if _, err := expfmt.MetricFamilyToText(writer, mf); err != nil {
			return "", fmt.Errorf("failed to export '%s': %w", *mf.Name, err)
		}
	}
	lines := strings.Split(writer.String(), "\n")
	linesFiltered := make([]string, 0, len(lines)/2)
	for _, ln := range lines {
		if strings.HasPrefix(ln, "#") {
			continue
		}
		if !includeZeroValues && strings.HasSuffix(ln, " 0") {
			continue
		}
		linesFiltered = append(linesFiltered, ln)
	}
	return strings.Join(linesFiltered, "\n"), nil
}

// Prefix is the prefix added to all metric names inside this factory
func (factory *MetricFactory) Prefix() string {
	return factory.namePrefix
}

func (factory *MetricFactory) concatNameAndLabels(name string, labelNames []string, leftmostLabelValues []string) (string, []string, []string) {
	if len(labelNames) < len(leftmostLabelValues) {
		logger.Panicf("length of labelNames (%s) should be equal or greater than length of leftmostLabelValues (%s)",
			strings.Join(labelNames, ","), strings.Join(leftmostLabelValues, ","))
	}
	fullName := factory.namePrefix + name
	allLabelNames := append(append([]string(nil), factory.parentLabelNames...), labelNames...)
	allLeftmostLabelValues := append(append([]string(nil), factory.parentLabelValues...), leftmostLabelValues...)
	return fullName, allLabelNames, allLeftmostLabelValues
}

func buildLabels(labelNames []string, leftmostLabelValues []string) map[string]string {
	if len(labelNames) < len(leftmostLabelValues) {
		logger.Panicf("length of labelNames (%s) should be equal or greater than length of leftmostLabelValues (%s)",
			strings.Join(labelNames, ","), strings.Join(leftmostLabelValues, ","))
	}
	labelMap := make(map[string]string, len(leftmostLabelValues))
	for i, value := range leftmostLabelValues {
		labelMap[labelNames[i]] = value
	}
	return labelMap
}
