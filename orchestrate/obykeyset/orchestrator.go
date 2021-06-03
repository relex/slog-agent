package obykeyset

import (
	"strings"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/orchestrate/obase"
	"github.com/relex/slog-agent/util"
)

// byKeySetOrchestrator is used to ensure fairer sharing of resource among logs of different key sets,
// e.g. using "level" as key field allows priority processing of error logs when there are massive amounts of debug logs before them
// A per-connection version has been tried and abandoned because a client may create a new connection after the old one dies, and both need to share info here
type byKeySetOrchestrator struct {
	logger         logger.Logger
	workerMap      *globalPipelineChannelMap // append-only global map of merged keys => worker channel
	keyLocators    []base.LogFieldLocator
	tagBuilder     *obase.TagBuilder // builder to construct tag from keys, used when protected by globalPipelineChannelMap's mutex
	metricFactory  *base.MetricFactory
	metricKeyNames []string
	launchWorkers  base.PipelineWorkersLauncher // start workers for new pipeline (one per key-set), invoked within globalPipelineChannelMap's mutex
}

// byKeySetOrchestratorChannel is created for each of input sessions or connections
// It holds local cache of pending logs to a set of real (global) workers used by this channel and flushes on demand
type byKeySetOrchestratorChannel struct {
	logger          logger.Logger
	workerMap       *localPipelineChannelMap // append-only locac cache of byKeySetOrchestrator.workerMap
	keySetExtractor base.FieldSetExtractor   // extractor to fetch keys from LogRecord(s)
	sendTimeout     *time.Timer
}

// NewOrchestrator creates an Orchestrator to distribute logs to different pipelines by unique combinations of key labels (key set)
func NewOrchestrator(parentLogger logger.Logger, schema base.LogSchema, keyFields []string, tagTemplate string,
	metricFactory *base.MetricFactory, launchWorkers base.PipelineWorkersLauncher, existingPipelineIDs []string) base.Orchestrator {
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
		metricFactory:  metricFactory,
		metricKeyNames: metricKeyNames,
		launchWorkers:  launchWorkers,
	}
	o.workerMap = newGlobalPipelineChannelMap(o.newWorker, closePipelineChannel, obase.NewPipelineChannelLocalBuffer)
	if len(existingPipelineIDs) > 0 {
		localMap := o.workerMap.MakeLocalMap()
		for _, pipelineID := range existingPipelineIDs {
			keys := strings.Split(pipelineID, ",")
			if len(keys) != len(keyFields) {
				ologger.Warnf("ignore malformed existing pipeline ID: %s", pipelineID)
				continue
			}
			localMap.GetOrCreate(keys)
		}
	}
	return o
}

func (o *byKeySetOrchestrator) NewChannel(id string) base.BufferReceiverChannel {
	plogger := o.logger.WithFields(logger.Fields{
		defs.LabelPart:   "channel",
		defs.LabelRemote: id,
	})
	return &byKeySetOrchestratorChannel{
		logger:          plogger,
		workerMap:       o.workerMap.MakeLocalMap(),
		keySetExtractor: *base.NewFieldSetExtractor(o.keyLocators),
		sendTimeout:     time.NewTimer(defs.IntermediateChannelTimeout),
	}
}

func (o *byKeySetOrchestrator) Destroy() {
	o.logger.Infof("destroying pipeline workers count=%d", util.PeekWaitGroup(o.workerMap.objectCounter))
	o.workerMap.Destroy()
	o.logger.Info("destroyed all pipeline workers")
}

// newWorker creates channel and pipeline workers for a new key-set, must be protected by global mutex
func (o *byKeySetOrchestrator) newWorker(keys []string, onStopped func()) chan<- []*base.LogRecord {
	tag := o.tagBuilder.Build(keys)
	workerID := strings.Join(keys, ",")
	inputChannel := make(chan []*base.LogRecord, defs.IntermediateBufferedChannelSize)
	pipelineLogger := o.logger.WithFields(logger.Fields{
		defs.LabelName: workerID,
	})
	pipelineLogger.Infof("new pipeline tag=%s", tag)
	pipelineMetricFactory := o.metricFactory.NewSubFactory(
		"process_",
		append([]string{"orchestrator"}, o.metricKeyNames...),
		append([]string{"byKeySet"}, keys...),
	)
	o.launchWorkers(pipelineLogger, tag, workerID, inputChannel, pipelineMetricFactory, onStopped)
	return inputChannel
}

// Accept accepts input logs from LogInput, the buffer is only usable within the function
func (oc *byKeySetOrchestratorChannel) Accept(buffer []*base.LogRecord) {
	now := time.Now()
	keySetExtractor := oc.keySetExtractor
	workerMap := oc.workerMap
	for _, record := range buffer {
		tempKeySet := keySetExtractor.Extract(record)
		cache := workerMap.GetOrCreate(tempKeySet)
		if cache.Append(record) {
			cache.Flush(now, oc.sendTimeout, oc.logger, tempKeySet)
		}
	}
}

// Tick renews internal timeout timer
func (oc *byKeySetOrchestratorChannel) Tick() {
	oc.flushAllLocalBuffers(false)
	util.ResetTimer(oc.sendTimeout, defs.IntermediateChannelTimeout)
}

// Close flushes all pending logs
func (oc *byKeySetOrchestratorChannel) Close() {
	oc.logger.Info("close")
	oc.flushAllLocalBuffers(true)
}

func (oc *byKeySetOrchestratorChannel) flushAllLocalBuffers(forceAll bool) {
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
