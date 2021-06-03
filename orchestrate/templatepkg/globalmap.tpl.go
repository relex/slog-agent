package templatepkg

import (
	"sync"

	"github.com/cheekybits/genny/generic"
	"github.com/relex/slog-agent/util"
)

// globalObjectType is the type of object in the global map
type globalObjectType generic.Type

// localWrapperType is the type of wrapper to enclose global object in local cache
// It can be the same as globalObjectType, if no local properties are needed
type localWrapperType generic.Type

type globalBaseNameObjectConstructor func(keys []string, onStopped func()) globalObjectType

type globalBaseNameObjectDestructor func(obj globalObjectType)

type localBaseNameWrapperConstructor func(obj globalObjectType) localWrapperType

// globalBaseNameMap keeps a global map of worker channels, each of the channel is linked to a background worker
// The channels can only be added, never removed or replaced, because their references are copied to local cache for fast access
type globalBaseNameMap struct {
	globalMap     map[string]globalObjectType
	globalMutex   *sync.Mutex                     // mutex to protect globalMap
	createObject  globalBaseNameObjectConstructor // func to create new global object, called within global mutex
	deleteObject  globalBaseNameObjectDestructor  // func to destroy global object
	wrapObject    localBaseNameWrapperConstructor // func to wrap global object in local cache
	objectCounter *sync.WaitGroup                 // global object counter, for finalization
}

// localBaseNameMap keeps local cache of global channel map for fast access, unprotected by mutex
type localBaseNameMap struct {
	localMap  map[string]localWrapperType // local cache of map, append only
	source    *globalBaseNameMap          // point to the source channelMap
	keyBuffer []byte                      // preallocated buffer to merge keys
}

// newGlobalBaseNameMap creates a globalBaseNameMap
func newGlobalBaseNameMap(create globalBaseNameObjectConstructor, delete globalBaseNameObjectDestructor, wrap localBaseNameWrapperConstructor) *globalBaseNameMap {
	return &globalBaseNameMap{
		globalMap:     make(map[string]globalObjectType, 2000),
		globalMutex:   &sync.Mutex{},
		createObject:  create,
		deleteObject:  delete,
		wrapObject:    wrap,
		objectCounter: &sync.WaitGroup{},
	}
}

// MakeLocalMap creates an empty local cache for a single log producer (e.g. a connection handled by one goroutine)
func (cm *globalBaseNameMap) MakeLocalMap() *localBaseNameMap {
	return &localBaseNameMap{
		localMap:  make(map[string]localWrapperType, 1000),
		source:    cm,
		keyBuffer: make([]byte, 0, 200),
	}
}

// Destroy closes all worker channels and wait for workers to finish
func (cm *globalBaseNameMap) Destroy() {
	cm.globalMutex.Lock()
	for _, obj := range cm.globalMap {
		cm.deleteObject(obj)
	}
	cm.globalMutex.Unlock()
	cm.objectCounter.Wait()
}

func (cm *globalBaseNameMap) getOrCreate(keys []string, mergedKey string) globalObjectType {
	cm.globalMutex.Lock()
	obj, found := cm.globalMap[mergedKey]
	if !found {
		cm.objectCounter.Add(1)
		obj = cm.createObject(keys, cm.objectCounter.Done)
		cm.globalMap[mergedKey] = obj
	}
	cm.globalMutex.Unlock()
	return obj
}

// LocalMap returns the local map. It shouild not be modified.
func (lm *localBaseNameMap) LocalMap() map[string]localWrapperType {
	return lm.localMap
}

// GetOrCreate gets or creates a channel-worker and returns local cache of it
// The given key-set is assumed to be transient and will be copied if it needs to enter global map
func (lm *localBaseNameMap) GetOrCreate(tempKeys []string) localWrapperType {
	tempMergedKey := lm.keyBuffer
	for _, tkey := range tempKeys {
		tempMergedKey = append(tempMergedKey, tkey...)
	}
	lm.keyBuffer = tempMergedKey[:0]
	// try to get existing cache by temp key, no new key string is created here
	if cache, found := lm.localMap[string(tempMergedKey)]; found {
		return cache
	}
	permKeys := util.DeepCopyStrings(tempKeys)
	permMergedKey := util.DeepCopyStringFromBytes(tempMergedKey)
	// pass copy of mergedKey here because it may be stored in global map
	newGlobalMap := lm.source.getOrCreate(permKeys, permMergedKey)
	newCache := lm.source.wrapObject(newGlobalMap)
	lm.localMap[permMergedKey] = newCache
	return newCache
}
