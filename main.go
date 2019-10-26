package main

import (
	"github.com/dwood15/mediaplayer/songplayer"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func init() {
	//This section is my (pitiful) attempt at keeping clean-boot performance reasonable
	runtime.GOMAXPROCS(3)

	if err := syscall.Setpriority(syscall.PRIO_PROCESS, 0x0, 19); err != nil {
		panic("failed setting process priority")
	}

	cfg := loadConfig()
	songplayer.SetLibraryDir(cfg.MusicDir)
	songplayer.SetPlaylistMaxSize(cfg.MaxPlaylistSize)
	go handleShutdown()
}

func main() {
	f, err := os.OpenFile("stderr.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	//Attempt to redirect panics and regular stderr messages to stderr.log
	_ = syscall.Dup2(int(f.Fd()), 2)

	go func() {
		//BeginPlaying enters into an infinite loop
		songplayer.GetLibrary().BeginPlaying()
	}()

	launchUI()
}

func handleShutdown() {
	// Handle graceful shutdown
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	//Indicate to the player that we're about to GO DOWN
	songplayer.PlayerSignal <- songplayer.SignalExit

	os.Exit(0)
}
