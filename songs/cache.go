package songs

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"os"
)

var c = cache.New(cache.NoExpiration, 0)
var lib *SongLibrary

const cacheName = "songlib.cache"

func init() {
	fp, err := os.OpenFile(cacheName, os.O_CREATE&os.O_RDWR, 0666)
	if err == nil {
		_ = c.Load(fp)
		return
	}

	if os.IsNotExist(err) {
		fp, err = os.Create(cacheName)
		if err != nil {
			panic(err)
		}

	} else {
		panic(err)
	}
}

func PersistLibCache() {
	c.Set(cacheName, lib, -1)

	fp, err := os.Create(cacheName)
	if err != nil {
		panic(err)
	}

	defer fp.Close()

	if err = c.Save(fp); err != nil {
		panic(err)
	}
}

func GetLibrary() *SongLibrary {
	if lib == nil {
		if l, found := c.Get(cacheName); found {
			lib = l.(*SongLibrary)
		}
	}

	if lib == nil {
		lib = &SongLibrary{}
		lib.LoadFromFiles()
		c.Set(cacheName, lib, -1)
		fmt.Println("loaded and persisted lib from cache")
	}

	if !lib.Pruned {
		lib.prune()
	}

	return lib
}
