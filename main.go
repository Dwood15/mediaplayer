package main

import (
	"encoding/json"
	"fmt"
	"github.com/dwood15/mediaplayer/songs"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type Config struct {
	MusicDir string `json:"music_dir"` //MusicDir is the directory where the
}

func handleShutdown() {
	// Handle graceful shutdown
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	fmt.Println("shut down signal received! saving library state to the cache, then exiting")
	songs.PersistLibCache()

	os.Exit(0)
}

func init() {
	runtime.GOMAXPROCS(4)

	prio, err := syscall.Getpriority(syscall.PRIO_PROCESS, 0x0)

	if err != nil {
		panic("err getting priority")
	}

	fmt.Printf("detected priority: %d\n", prio)

	fmt.Println("Setting priority lower")
	err = syscall.Setpriority(syscall.PRIO_PROCESS, 0x0, 19)
	if err != nil {
		panic("failed setting process priority")
	}

	loadConfig()
	songs.SetLibraryDir(cfg.MusicDir)
}

func main() {
	start := time.Now()

	l := songs.GetLibrary()
	go handleShutdown()

	songs.PersistLibCache()

	fmt.Printf("Loaded library in: %v\n", time.Since(start))
	fmt.Printf("Total song time loaded: %v\n", l.TotalTime)

	for {
		l.Play()
		songs.PersistLibCache()
	}
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
