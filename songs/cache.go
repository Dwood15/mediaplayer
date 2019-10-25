package songs

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

var lib *SongLibrary

const cacheName = "songlib.cache"

func (lib *SongLibrary) persistSelf() {
	res, err := json.MarshalIndent(lib, "", "  ")
	if err != nil {
		panic(err)
	}

	fp, err := os.Create(cacheName)
	if err != nil {
		panic(err)
	}

	defer fp.Close()

	if err := ioutil.WriteFile(cacheName, res, 0666); err != nil {
		panic(err)
	}
}

//GetLibrary attempts to load the SongLibrary, for media-playing functionality.
func GetLibrary() *SongLibrary {
	defer func() {
		if n := len(lib.Songs); maxSize > n {
			maxSize = n
		}
		lib.computeScores()
	}()

	if lib != nil {
		return lib
	}

	lib = &SongLibrary{}

	res, err := ioutil.ReadFile(cacheName)
	if err == nil {
		if err = json.Unmarshal(res, lib); err == nil {
			return lib
		}
	}

	if !os.IsNotExist(err) {
		panic(err)
	}

	lib.LoadFromFiles()

	return lib
}
