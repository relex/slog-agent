package obase

import (
	"sync"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// Distributor distributes log records to fixed numbers of child workers for parallel processing
// Distribution is done by ex: first 100 logs to 1st pipeline, next 100 logs to 2nd pipeline, and so on
// The child workers must be OrderedLogProcessingWorker, which uses locks to keep original order when calling LogChunkAccepter
type Distributor struct {
	bsupport.PipelineWorkerBaseForLogRecords
	children     []chan<- base.OrderedLogBuffer
	childCounter *sync.WaitGroup
	currentBatch *distributionBatch
}

type distributionBatch struct {
	childIndex    int
	previousMutex *sync.Mutex
	currentMutex  *sync.Mutex
	numRecords    int
	numBytes      int
}

// NewDistributor creates Distributor
func NewDistributor(parentLogger logger.Logger, input <-chan []*base.LogRecord, tag string, numWorkers int,
	metricCreator promreg.MetricCreator, launchChildPipeline base.OrderedPipelineWorkersLauncher) *Distributor {
	dlogger := parentLogger.WithField(defs.LabelComponent, "ParallelDistributor")
	children := make([]chan<- base.OrderedLogBuffer, numWorkers)
	childCounter := &sync.WaitGroup{}
	for i := range children {
		childCounter.Add(1)
		childChannel := make(chan base.OrderedLogBuffer, defs.IntermediateBufferedChannelSize)
		launchChildPipeline(dlogger.WithField("child", i), tag, i, childChannel, metricCreator, childCounter.Done)
		children[i] = childChannel
	}
	prevMutex := &sync.Mutex{}
	currMutex := &sync.Mutex{}
	currMutex.Lock()
	dist := &Distributor{
		PipelineWorkerBaseForLogRecords: bsupport.NewPipelineWorkerBaseForLogRecords(dlogger, input, false),
		children:                        children,
		childCounter:                    childCounter,
		currentBatch: &distributionBatch{
			childIndex:    0,
			previousMutex: prevMutex,
			currentMutex:  currMutex,
			numRecords:    0,
			numBytes:      0,
		},
	}
	dist.InitInternal(dist.onInput, nil, dist.onStop)
	return dist
}

func (dist *Distributor) onInput(records []*base.LogRecord, timeout <-chan time.Time) {
	batchInfo := dist.currentBatch
	batchInfo.numRecords += len(records)
	batchInfo.numBytes += bsupport.SumLogRecordsLength(records)
	var lastInBatch bool
	if batchInfo.numRecords >= defs.ParallelizationBufferMaxNumLogs ||
		batchInfo.numBytes >= defs.ParallelizationBufferMaxTotalBytes {
		lastInBatch = true
	} else {
		lastInBatch = false
	}
	dist.Logger().Debugf("relay %d records", len(records))
	orderedBuffer := base.OrderedLogBuffer{
		Records:  records,
		Previous: batchInfo.previousMutex,
		Current:  batchInfo.currentMutex,
		IsLast:   lastInBatch,
	}
	select {
	case dist.children[batchInfo.childIndex] <- orderedBuffer:
		// TODO: update metrics
		break
	case <-timeout:
		dist.Logger().Errorf("timeout relaying: %d records for %d", len(orderedBuffer.Records), batchInfo.childIndex)
		break
	}
	if lastInBatch {
		dist.nextPipeline()
	}
}

func (dist *Distributor) onStop(timeout <-chan time.Time) {
	dist.Logger().Infof("stopping child pipelines, count=%d", util.PeekWaitGroup(dist.childCounter))
	for _, ch := range dist.children {
		close(ch)
	}
	dist.childCounter.Wait()
	dist.Logger().Info("stopped child pipelines")
}

func (dist *Distributor) nextPipeline() {
	old := dist.currentBatch
	newMutex := &sync.Mutex{}
	newMutex.Lock()
	new := &distributionBatch{
		childIndex:    (old.childIndex + 1) % len(dist.children),
		previousMutex: old.currentMutex,
		currentMutex:  newMutex,
		numRecords:    0,
		numBytes:      0,
	}
	dist.Logger().Debugf("switch from pipeline %d to %d", old.childIndex, new.childIndex)
	dist.currentBatch = new
}
