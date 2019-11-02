package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/dwood15/mediaplayer/sockets"
	"github.com/dwood15/mediaplayer/songplayer"
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

const ssIdx = int(unsafe.Offsetof(songplayer.PlayingSong{}.SongScore))
const slIdx = int(unsafe.Offsetof(songplayer.PlayingSong{}.SongLength))
const sNIdx = int(unsafe.Offsetof(songplayer.PlayingSong{}.CurrentSong))
const szOf = int(unsafe.Sizeof(songplayer.PlayingSong{}))

func isClosed(fd int) bool {
	b := make([]byte, 1)

	n, _, err := unix.Recvfrom(fd, b, unix.MSG_DONTWAIT|unix.MSG_PEEK)

	if err != nil && !(err.(unix.Errno)).Temporary() {
		fmt.Printf("connection closed potentially detected! err:\n\t[%v]\n", err)
		return true
	}

	return n == 0
}

func amServ(sU chan songplayer.PlayingSong) {
	fmt.Println("Client not found, assuming we're the server.")

	addr := &unix.SockaddrUnix{Name: sockName}

	srv := sockets.Server{
		SockAddr: addr,
		OnConnect: func(cFD int, done chan bool) {
			fmt.Println("client connection made!")

			bytesToSend := make([]byte, unsafe.Sizeof(songplayer.PlayingSong{})+32)

			for {
				select {
				case ss := <-sU:
					binary.PutUvarint(bytesToSend[ssIdx:ssIdx+10:szOf], ss.SongScore)
					binary.PutVarint(bytesToSend[slIdx:slIdx+10:szOf], int64(ss.SongLength))
					n := copy(bytesToSend[sNIdx:], ss.CurrentSong)

					fmt.Printf("num bytes Copied to toSend slice: [%d]\n", n)

					if n, err := unix.Write(cFD, bytesToSend); err != nil {
						fmt.Printf("non-nil err when attempting sendTo: %v\n", err)
					} else {
						fmt.Printf("[%d] Bytes written in our attempt to send\n", n)
					}
				case <-done:
					fmt.Println("close signal detected, closing connection")
					return
				default:
					if isClosed(cFD) {
						fmt.Println("client connection closed. returning from OnConnect")
						return
					}
				}
			}
		},
	}

	if err := srv.LaunchServer(); err != nil {
		panic("launchsrvr: " + err.Error())
	}

	fmt.Println("loading library and preparing to play")
	fmt.Println("server will wait for incoming connection before playing")
	//BeginPlaying enters into an infinite loop
	songplayer.GetLibrary(sU).BeginPlaying()
}

func amUI(fd int) {

}

var uiInput = make(chan int64)
var serverSongUpdate = make(chan songplayer.PlayingSong)

func main() {
	c := sockets.Client{Addr: &unix.SockaddrUnix{Name: sockName}}

	fmt.Println("attempting to launch ui client")
	if err := c.LaunchClient(uiInput, serverSongUpdate); err != nil {
		amServ(serverSongUpdate)
		os.Exit(0)
	}

	f, err := os.OpenFile("stderr.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	//Attempt to redirect panics and regular stderr messages to stderr.log
	_ = syscall.Dup2(int(f.Fd()), 2)

	time.Sleep(5 * time.Second)

	launchUI(uiInput, serverSongUpdate)
}

func handleShutdown() {
	// Handle graceful shutdown
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	//Indicate to the player that we're about to GO DOWN
	uiInput <- songplayer.SignalExit

	os.Exit(0)
}
