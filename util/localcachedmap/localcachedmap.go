// Package localcachedmap provides a map with thread-local caches (copy-on-reference)
package localcachedmap

import (
	"sync"

	"github.com/relex/slog-agent/util"
)

// GlobalObjectConstructor defines a function that creates a new global item in GlobalCachedMap
type GlobalObjectConstructor[G any] func(keys []string, onStopped func()) G

// GlobalObjectDestructor defines a function that destroys a global item in GlobalCachedMap
type GlobalObjectDestructor[G any] func(obj G)

// LocalWrapperConstructor defines a function that creates a local wrapper for a global item in GlobalCachedMap
type LocalWrapperConstructor[G any, L any] func(obj G) L

// GlobalCachedMap keeps a global map of objects that can be looked up in thread-local cache
//
// The objects can only be added, never removed or replaced, because their references are copied to local cache for fast access
type GlobalCachedMap[G any, L any] struct {
	globalMap     map[string]G
	globalMutex   *sync.Mutex                   // mutex to protect globalMap
	createObject  GlobalObjectConstructor[G]    // func to create new global object, called within global mutex
	deleteObject  GlobalObjectDestructor[G]     // func to destroy global object
	wrapObject    LocalWrapperConstructor[G, L] // func to wrap global object in local cache
	objectCounter *sync.WaitGroup               // global object counter, for finalization
}

// NewGlobalMap creates a globalMap
func NewGlobalMap[G any, L any](
	create GlobalObjectConstructor[G],
	delete GlobalObjectDestructor[G],
	wrap LocalWrapperConstructor[G, L],
) *GlobalCachedMap[G, L] {

	return &GlobalCachedMap[G, L]{
		globalMap:     make(map[string]G, 2000),
		globalMutex:   &sync.Mutex{},
		createObject:  create,
		deleteObject:  delete,
		wrapObject:    wrap,
		objectCounter: &sync.WaitGroup{},
	}
}

// MakeLocalMap creates an empty local cache for a single thread (e.g. a connection handled by one goroutine)
//
// All access to the underlying map should be done through LocalCachedMap for performance
func (gm *GlobalCachedMap[G, L]) MakeLocalMap() *LocalCachedMap[G, L] {
	return &LocalCachedMap[G, L]{
		localMap:  make(map[string]L, 1000),
		source:    gm,
		keyBuffer: make([]byte, 0, 200),
	}
}

func (gm *GlobalCachedMap[G, L]) PeekNumObjects() int {
	return util.PeekWaitGroup(gm.objectCounter)
}

// Destroy closes all worker channels and wait for workers to finish
func (gm *GlobalCachedMap[G, L]) Destroy() {
	gm.globalMutex.Lock()
	for _, obj := range gm.globalMap {
		gm.deleteObject(obj)
	}
	gm.globalMutex.Unlock()
	gm.objectCounter.Wait()
}

func (gm *GlobalCachedMap[G, L]) getOrCreate(keys []string, mergedKey string) G {
	gm.globalMutex.Lock()
	obj, found := gm.globalMap[mergedKey]
	if !found {
		gm.objectCounter.Add(1)
		obj = gm.createObject(keys, gm.objectCounter.Done)
		gm.globalMap[mergedKey] = obj
	}
	gm.globalMutex.Unlock()
	return obj
}

// LocalCachedMap keeps local cache of global map for fast access, unprotected by mutex
type LocalCachedMap[G any, L any] struct {
	localMap  map[string]L           // local cache of map, append only
	source    *GlobalCachedMap[G, L] // point to the source channelMap
	keyBuffer []byte                 // preallocated buffer to merge keys
}

// LocalMap returns the local map. It shouild not be modified.
func (lm *LocalCachedMap[G, L]) LocalMap() map[string]L {
	return lm.localMap
}

// GetOrCreate gets or creates a global object and returns the local cache of it
//
// The given tempKeys is assumed to be transient and will be copied if it needs to be stored in the global map
func (lm *LocalCachedMap[G, L]) GetOrCreate(tempKeys []string) L {
	tempMergedKey := lm.keyBuffer
	for _, tkey := range tempKeys {
		tempMergedKey = append(tempMergedKey, tkey...)
	}
	lm.keyBuffer = tempMergedKey[:0]

	// try to get existing cache by temp key, no new key string is created here
	if cache, found := lm.localMap[string(tempMergedKey)]; found {
		return cache
	}

	// make permanent copy of keys here so that they may be used globally
	permanentKeys := util.DeepCopyStrings(tempKeys)
	permanentMergedKey := util.DeepCopyStringFromBytes(tempMergedKey)

	newGlobalObject := lm.source.getOrCreate(permanentKeys, permanentMergedKey)
	newLocalCache := lm.source.wrapObject(newGlobalObject)
	lm.localMap[permanentMergedKey] = newLocalCache
	return newLocalCache
}
