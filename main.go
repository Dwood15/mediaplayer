package main

import (
	"encoding/json"
	"fmt"
	"github.com/dwood15/mediaplayer/songs"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type config struct {
	MusicDir        string `json:"music_dir"` //MusicDir is the directory where the
	MaxPlaylistSize int    `json:"max_playlist_size"`
}

func handleShutdown() {
	// Handle graceful shutdown
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	//fmt.Println("shut down signal received! saving library state to the cache, then exiting")
	songs.PersistLibCache()

	os.Exit(0)
}

func init() {
	runtime.GOMAXPROCS(3)

	if err := syscall.Setpriority(syscall.PRIO_PROCESS, 0x0, 19); err != nil {
		panic("failed setting process priority")
	}

	loadConfig()
	songs.SetLibraryDir(cfg.MusicDir)
	songs.SetPlaylistMaxSize(cfg.MaxPlaylistSize)
	go handleShutdown()
}

func fmtDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d", m, s)
}

var (
	app  = tview.NewApplication()
	view = tview.NewBox().SetDrawFunc(drawTime)
)

var songTime time.Duration

func drawTime(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	timeStr := fmtDuration(songTime)
	tview.Print(screen, timeStr, x, height/2, width, tview.AlignCenter, tcell.ColorDarkBlue)
	return 0, 0, 0, 0
}

func refresh() {
	for {
		select {
		case songTime = <-songs.SongTime:
			app.Draw()
		case <-songs.SongState:

		}
	}
}

func main() {
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	//Redirect panic and Stderr to the file
	os.Stderr = f
	_ = syscall.Dup2(int(f.Fd()), 2)

	defer f.Close()

	l := songs.GetLibrary()

	go func() {
		for {
			songs.PersistLibCache()
			l.Play()
		}
	}()

	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEsc {
			app.Stop()
		}
		return e
	})

	view.SetBackgroundColor(tcell.ColorBlack)
	view.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		songs.HotkeyEvent <- e
		return e
	})

	go refresh()
	if err := app.SetRoot(view, false).Run(); err != nil {
		panic(err)
	}
}

var cfg config

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

	if err = json.Unmarshal(b, &cfg); err != nil {
		panic(err)
	}
}
