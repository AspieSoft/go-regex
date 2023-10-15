package common

import (
	"sync"
	"time"
)

type CacheMap[T any] struct {
	value map[string]T
	err map[string]error
	lastUse map[string]time.Time
	mu sync.Mutex
	null T
}

func NewCache[T any]() CacheMap[T] {
	return CacheMap[T]{
		value: map[string]T{},
		err: map[string]error{},
		lastUse: map[string]time.Time{},
	}
}

// get returns a value or an error if it exists
//
// if the object key does not exist, it will return both a nil/zero value (of the relevant type) and nil error
func (cache *CacheMap[T]) Get(key string) (T, error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if err, ok := cache.err[key]; ok {
		cache.lastUse[key] = time.Now()
		return cache.null, err
	}else if val, ok := cache.value[key]; ok {
		cache.lastUse[key] = time.Now()
		return val, nil
	}

	return cache.null, nil
}

// set sets or adds a new key with either a value, or an error
func (cache *CacheMap[T]) Set(key string, value T, err error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if err != nil {
		cache.err[key] = err
		delete(cache.value, key)
		cache.lastUse[key] = time.Now()
	}else{
		cache.value[key] = value
		delete(cache.err, key)
		cache.lastUse[key] = time.Now()
	}
}

// delOld removes old cache items
func (cache *CacheMap[T]) DelOld(cacheTime time.Duration){
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if cacheTime == 0 {
		for key := range cache.lastUse {
			delete(cache.value, key)
			delete(cache.err, key)
			delete(cache.lastUse, key)
		}
		return
	}

	now := time.Now().UnixNano()

	for key, lastUse := range cache.lastUse {
		if now - lastUse.UnixNano() > int64(cacheTime) {
			delete(cache.value, key)
			delete(cache.err, key)
			delete(cache.lastUse, key)
		}
	}
}
