package obykeyset

import (
	"strings"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/orchestrate/obase"
	"github.com/relex/slog-agent/util"
	"github.com/relex/slog-agent/util/localcachedmap"
)

// byKeySetOrchestrator is used to ensure fairer sharing of resource among logs of different key sets,
// e.g. using "level" as key field allows priority processing of error logs when there are massive amounts of debug logs before them
// A per-connection version has been tried and abandoned because a client may create a new connection after the old one dies, and both need to share info here
type byKeySetOrchestrator struct {
	logger         logger.Logger
	workerMap      *globalPipelineChannelMap // append-only global map of merged keys => worker channel
	keyLocators    []base.LogFieldLocator
	tagBuilder     *obase.TagBuilder // builder to construct tag from keys, used when protected by globalPipelineChannelMap's mutex
	metricCreator  promreg.MetricCreator
	metricKeyNames []string
	launchWorkers  base.PipelineWorkersLauncher // start workers for new pipeline (one per key-set), invoked within globalPipelineChannelMap's mutex
}

// byKeySetOrchestratorSink is created for each of input sessions or connections
// It holds local cache of pending logs to a set of real (global) workers used by this channel and flushes on demand
type byKeySetOrchestratorSink struct {
	logger          logger.Logger
	workerMap       *localPipelineChannelMap // append-only locac cache of byKeySetOrchestrator.workerMap
	keySetExtractor base.FieldSetExtractor   // extractor to fetch keys from LogRecord(s)
	sendTimeout     *time.Timer
}

// NewOrchestrator creates an Orchestrator to distribute logs to different pipelines by unique combinations of key labels (key set)
func NewOrchestrator(parentLogger logger.Logger, schema base.LogSchema, keyFields []string, tagTemplate string,
	metricCreator promreg.MetricCreator, launchWorkers base.PipelineWorkersLauncher, existingPipelineIDs []string) base.Orchestrator {
	ologger := parentLogger.WithField(defs.LabelComponent, "ByKeySetOrchestrator")
	keyLocators, lerr := schema.CreateFieldLocators(keyFields)
	if lerr != nil {
		ologger.Panicf("keyFields: %s", lerr.Error())
	}
	tagBuilder, terr := obase.NewTagBuilder(tagTemplate, keyFields)
	if terr != nil {
		ologger.Panicf("tagTemplate: %s", terr.Error())
	}
	metricKeyNames := make([]string, len(keyFields))
	for i, key := range keyFields {
		metricKeyNames[i] = "key_" + key
	}
	o := &byKeySetOrchestrator{
		logger:         ologger,
		workerMap:      nil,
		keyLocators:    keyLocators,
		tagBuilder:     tagBuilder,
		metricCreator:  metricCreator,
		metricKeyNames: metricKeyNames,
		launchWorkers:  launchWorkers,
	}
	o.workerMap = localcachedmap.NewGlobalMap(o.newWorker, closePipelineChannel, obase.NewPipelineChannelLocalBuffer)

	if len(existingPipelineIDs) > 0 {
		localMap := o.workerMap.MakeLocalMap()
		onCreating := func([]string) {}
		for _, pipelineID := range existingPipelineIDs {
			keys := strings.Split(pipelineID, ",")
			if len(keys) != len(keyFields) {
				// FIXME: deal with new keys, shorter old keys should be okay
				ologger.Warnf("ignore malformed existing pipeline ID: %s", pipelineID)
				continue
			}
			localMap.GetOrCreate(keys, onCreating)
		}
	}
	return o
}

func (o *byKeySetOrchestrator) NewSink(clientAddress string, clientNumber base.ClientNumber) base.BufferReceiverSink {
	return &byKeySetOrchestratorSink{
		logger:          base.NewSinkLogger(o.logger, clientAddress, clientNumber),
		workerMap:       o.workerMap.MakeLocalMap(),
		keySetExtractor: *base.NewFieldSetExtractor(o.keyLocators),
		sendTimeout:     time.NewTimer(defs.IntermediateChannelTimeout),
	}
}

func (o *byKeySetOrchestrator) Shutdown() {
	o.logger.Infof("shutting down pipeline workers count=%d", o.workerMap.PeekNumObjects())
	o.workerMap.Destroy()
	o.logger.Info("shut down all pipeline workers")
}

// newWorker creates channel and pipeline workers for a new key-set, must be protected by global mutex
func (o *byKeySetOrchestrator) newWorker(keys []string, onStopped func()) chan<- []*base.LogRecord {
	tag := o.tagBuilder.Build(keys)
	workerID := strings.Join(keys, ",")
	inputChannel := make(chan []*base.LogRecord, defs.IntermediateBufferedChannelSize)
	pipelineLogger := o.logger.WithField(defs.LabelName, workerID)
	pipelineLogger.Infof("new pipeline tag=%s", tag)
	pipelineMetricCreator := o.metricCreator.AddOrGetPrefix(
		"process_",
		append([]string{"orchestrator"}, o.metricKeyNames...),
		append([]string{"byKeySet"}, keys...),
	)
	o.launchWorkers(pipelineLogger, tag, workerID, inputChannel, pipelineMetricCreator, onStopped)
	return inputChannel
}

// Accept accepts input logs from LogInput, the buffer is only usable within the function
func (oc *byKeySetOrchestratorSink) Accept(buffer []*base.LogRecord) {
	now := time.Now()
	keySetExtractor := oc.keySetExtractor
	workerMap := oc.workerMap
	for _, record := range buffer {
		tempKeySet := keySetExtractor.Extract(record)
		cache := workerMap.GetOrCreate(tempKeySet, oc.onNewLinkToPipeline)
		if cache.Append(record) {
			cache.Flush(now, oc.sendTimeout, oc.logger, tempKeySet)
		}
	}
}

// Tick renews internal timeout timer
func (oc *byKeySetOrchestratorSink) Tick() {
	oc.flushAllLocalBuffers(false)
	util.ResetTimer(oc.sendTimeout, defs.IntermediateChannelTimeout)
}

// Close flushes all pending logs
func (oc *byKeySetOrchestratorSink) Close() {
	oc.logger.Info("close")
	oc.flushAllLocalBuffers(true)
}

func (oc *byKeySetOrchestratorSink) onNewLinkToPipeline(permKeys []string) {
	workerID := strings.Join(permKeys, ",")
	oc.logger.WithField(defs.LabelName, workerID).Info("creating new link from input to pipeline worker")
}

func (oc *byKeySetOrchestratorSink) flushAllLocalBuffers(forceAll bool) {
	now := time.Now()
	for mergedKey, cache := range oc.workerMap.LocalMap() {
		if len(cache.PendingLogs) == 0 {
			continue
		}
		if !forceAll && now.Sub(cache.LastFlushTime) < defs.IntermediateFlushInterval {
			continue
		}
		cache.Flush(now, oc.sendTimeout, oc.logger, mergedKey)
	}
}
