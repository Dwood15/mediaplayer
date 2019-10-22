package main

import (
	"encoding/json"
	"fmt"
	"github.com/dwood15/mediaplayer/songs"
	"io/ioutil"
	"os"
	"time"
)

type Config struct {
	MusicDir string `json:"music_dir"` //MusicDir is the directory where the
}

func init() {
	loadConfig()
	songs.SetLibraryDir(cfg.MusicDir)
}

func main() {
	start := time.Now()

	l := songs.GetLibrary()

	fmt.Printf("Loaded library in: %v\n", time.Since(start))

	l.Play()

	fmt.Printf("Total song time loaded: %v\n", l.TotalTime)

	l.Play()

	songs.PersistLibCache()

	fmt.Println("playing complete")
}

var cfg Config

func loadConfig() {
	f, err := os.Open("config.json")

	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}

		h, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}

		cfg.MusicDir = h + "/Music"

		f, err := os.Create("config.json")
		if err != nil {
			panic(err)
		}

		newConfig, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			panic(err)
		}

		if _, err = f.Write(newConfig); err != nil {
			panic(err)
		}

		f.Close()
		return
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	f.Close()

	err = json.Unmarshal(b, &cfg)
	if err != nil {
		panic(err)
	}

	fmt.Println("config found and loaded, music dir: " + cfg.MusicDir)
}
