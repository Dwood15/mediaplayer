// Package litecache provides interfaces for threadsafe, mildly-generic, in-memory caches
//These caches are intended to maximize performance as much as possible by helping to
//minimize expensive operations such as locking OS calls (File Reads, Network calls, et cetera)
package litecache

import (
	"time"
)

type (
	// Cache provides the public implementations for threadsafe object storage and
	//accesses. In order to avoid problems with dereferences and copies of the cache
	//structs, it only contains a pointer to the "actual" cache structure
	Cache struct {
		pc *protectedCache
	}

	// EntryUpdater the function passed to CreateCache, which is called on every cache
	//miss.
	EntryUpdater func(string) interface{}
)

// CreateCache allocates a cache and adds a reference to it to the pool of caches for regular cleaning.
//The new Cache is registerd with a background cachePooler that regularly cleans out expired entries.
//If an Expired entry was accessed at least once since the last cleaning time, the cleaner will update
//the entry. If the expired entry was accessed >=once, it will be removed and looked up next read.
//
//The updater function is expected to be coroutine/thread-safe and non-nil
func CreateCache(expireRate time.Duration, updater EntryUpdater) *Cache {
	if updater == nil {
		panic("the updater-func be a non-nil reference to a EntryUpdater")
	}

	c := &Cache{
		pc: &protectedCache{
			expireRate: expireRate,
			data:       map[string]expirable{},
			updater:    updater,
		},
	}

	//register the cache so expired entriesthe cleaner
	p.addCache(c.pc)

	return c
}

// Retrieve a value from the cache.
func (c *Cache) Retrieve(key string) (val interface{}) {
	//pluck from the cache
	c.pc.mu.RLock()
	cached, ok := c.pc.data[key]
	c.pc.mu.RUnlock()

	if val = cached.value; ok && !cached.neverAccessed {
		return
	}

	return c.pc.entryUpdate(key, cached, !ok)
}

//Implode inactivates the cache from the eager cleaning and eager updating processes.
//Once Implode is called, do CreateCache MUST be called again.
//SUBSEQUENT USES WILL PANIC
func (c *Cache) Implode() {
	if c == nil {
		return
	}
	c.pc.implode()
}
