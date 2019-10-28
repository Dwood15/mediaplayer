package main

import (
	"fmt"
	"github.com/dwood15/mediaplayer/sockets"
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

const sockName = "/tmp/mediaplayer.sock"


func amServ() {
	fmt.Println("Client not found, assuming we're the server.")

	srv := sockets.Server{
		SockName: sockName,
		OnConnect: func(cFD int, done chan bool) {
			fmt.Println("client connection detected")

			for {
				select {
				case <-done:
					fmt.Println("close signal detected, closing connection")
					return
				}
			}
		},
	}

	if err := srv.LaunchServer(); err != nil {
		panic("launchrvr: " + err.Error())
	}

	go func() {
		//BeginPlaying enters into an infinite loop
		songplayer.GetLibrary().BeginPlaying()
	}()
}

func amUI(fd int) {

}

func main() {
	fd := sockets.OpenClientfd(sockName)
	if fd == -1 {
		amServ()
		os.Exit(0)
	}

	f, err := os.OpenFile("stderr.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	//Attempt to redirect panics and regular stderr messages to stderr.log
	_ = syscall.Dup2(int(f.Fd()), 2)


	launchUI(fd)
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
