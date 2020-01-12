package litecache

import (
	"sync"
	"time"
)

type (
	//how we maintain
	cachePooler struct {
		mu        sync.RWMutex
		cleanRate time.Duration

		//pointers for concurrent accesses
		pool []*protectedCache
	}

	// encapsulate cache entries and indicate when it should expire
	expirable struct {
		neverAccessed bool
		expiresAt     time.Time
		value         interface{}
	}

	// protectedCache is the ACTUAL implementation of the cache mechanism
	protectedCache struct {
		mu         sync.RWMutex
		expireRate time.Duration
		data       map[string]expirable
		updater    func(string) interface{}
		poolIndex  int
	}
)

var p = cachePooler{
	cleanRate: 2 * time.Minute,
}

func init() {
	go func() {
		for {
			//Initiate the cache clean on arbitrary intervals
			//Slow cleaning in order to avoid unnecessarily burning CPU
			time.Sleep(p.cleanRate)
			p.mu.RLock()
			for i := 0; i < len(p.pool); i++ {
				if (p.pool)[i] == nil {
					continue
				}
				(p.pool)[i].processExpired()
			}
			p.mu.RUnlock()
		}
	}()
}

func (cp *cachePooler) addCache(c *protectedCache) {
	cp.mu.Lock()
	c.poolIndex = len(cp.pool)
	cp.pool = append(cp.pool, c)
	cp.mu.Unlock()
}

func (cp *cachePooler) removeCache(c *protectedCache) {
	if c == nil || c.poolIndex < 0 {
		return
	}

	p.mu.Lock()
	if cp.pool[c.poolIndex] != c {
		panic("cache poolIndex doesn't map to the same pointer as was passed in")
	}

	cp.pool[c.poolIndex] = nil
	c.poolIndex = -1
	p.mu.Unlock()
}

//only relevant to the cachePool
func (c *protectedCache) processExpired() {
	//Faster to acquire the write lock throughout the delete process than
	//to acquire locks individually for each delete
	c.mu.Lock()
	var wg sync.WaitGroup
	n := time.Now()

	for key, entry := range c.data {
		if !entry.expiresAt.Before(n) {
			continue
		}

		if entry.neverAccessed {
			delete(c.data, key)
			continue
		}

		wg.Add(1)
		//TODO: Chunk these ops so only a few are launched in goroutines at a time?
		go func(k string, n time.Time) {
			c.data[k] = expirable{
				value:         c.updater(k),
				neverAccessed: true,
				expiresAt:     n.Add(c.expireRate),
			}
			wg.Done()
		}(key, n)
	}
	wg.Wait()
	c.mu.Unlock()
}

func (c *protectedCache) implode() {
	c.mu.Lock()
	c.data = nil
	c.updater = nil
	p.removeCache(c)
	c.mu.Unlock()
}

func (c *protectedCache) entryUpdate(key string, entry expirable, callUpdater bool) expirable {
	c.mu.Lock()

	if entry.neverAccessed {
		entry.neverAccessed = false
	}

	if callUpdater {
		entry.expiresAt = time.Now().Add(c.expireRate)
		entry.value = c.updater(key)
	}

	c.data[key] = entry
	c.mu.Unlock()

	return entry
}
