package main

import (
	"github.com/dwood15/mediaplayer/songs"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

func init() {
	//This section is my (pitiful) attempt at keeping clean-boot performance reasonable
	runtime.GOMAXPROCS(3)

	if err := syscall.Setpriority(syscall.PRIO_PROCESS, 0x0, 19); err != nil {
		panic("failed setting process priority")
	}

	cfg := loadConfig()
	songs.SetLibraryDir(cfg.MusicDir)
	songs.SetPlaylistMaxSize(cfg.MaxPlaylistSize)
	go handleShutdown()
}

func main() {
	f, err := os.OpenFile("stderr.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	//Attempt to redirect panics and regular stderr messages to stderr.log
	os.Stderr = f
	_ = syscall.Dup2(int(f.Fd()), 2)

	defer f.Close()

	l := songs.GetLibrary()

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		//BeginPlaying enters into an infinite loop
		l.BeginPlaying()
		wg.Done()
	}()

	launchUI()
	//Even with the ui down, we need to wait for the library to close
	wg.Wait()
}

func handleShutdown() {
	// Handle graceful shutdown
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	//Indicate to the player that we're about to GO DOWN
	songs.PlayerSignal <- songs.SignalExit

	os.Exit(0)
}
