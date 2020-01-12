package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
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

const stIdx = int(unsafe.Offsetof(songplayer.PlayingSong{}.SongTime))
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

func amServ() {
	fmt.Println("Client not found, assuming we're the server.")

	addr := &unix.SockaddrUnix{Name: sockName}

	srv := sockets.Server{
		SockAddr: addr,
		OnConnect: func(cFD int, done chan bool) {
			fmt.Println("client connection made!")

			for {
				select {
				case ss := <-songplayer.SongState:


					//binary.PutVarint(bytesToSend[stIdx:stIdx+10:szOf], int64(ss.SongTime))
					//binary.PutUvarint(bytesToSend[ssIdx:ssIdx+10:szOf], ss.SongScore)
					//binary.PutVarint(bytesToSend[slIdx:slIdx+10:szOf], int64(ss.SongLength))
					//
					//copy(bytesToSend[sNIdx:], ss.CurrentSong)

					//fmt.Printf("num bytes Copied to toSend slice: [%d]\n", n)

					/* QUARANTINE SECTION */
					//THIS SECTION OF CODE IS INCREDIBLY FINNICKY, AND I HAVEN'T
					// SPENT TIME TO FIGURE OUT HOW TO MAKE IT ROBUST...
					// CONSIDER WRITING SOME UNIT TESTS OR MAKING INCREDIBLY MINOR
					// CHANGES BETWEEN TESTS
					bytesToSend, _ := json.Marshal(ss)

					if len(bytesToSend) > szOf + 128 {
						fmt.Println("WARNING: NUMBER OF BYTES LARGER THAN CLIENT EXPECTS: ", len(bytesToSend))
					}

					if _, err := unix.Write(cFD, bytesToSend); err != nil {
						fmt.Printf("non-nil err when attempting sendTo: %v\n", err)
					}
					//SEE ALSO: sockets/client.go
					/* QUARANTINE SECTION */

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

	go func() {
		if err := srv.LaunchServer(); err != nil {
			panic("launchsrvr: " + err.Error())
		}
	}()

	time.Sleep(1 * time.Second)
	fmt.Println("loading library and preparing to play")
	fmt.Println("server will wait for incoming connection before playing")
	//BeginPlaying enters into an infinite loop
	songplayer.GetLibrary().BeginPlaying()
}

var uiInput = make(chan int64)
var state = new(atomic.Value)

func main() {
	state.Store(songplayer.PlayingSong{})

	c := sockets.Client{
		Addr:            &unix.SockaddrUnix{Name: sockName},
		ServerSongState: state,
	}

	fmt.Println("attempting to launch ui client")
	if err := c.LaunchClient(uiInput); err != nil {
		amServ()
		os.Exit(0)
	}

	time.Sleep(7 * time.Second)

	f, err := os.OpenFile("stderr.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	//Attempt to redirect panics and regular stderr messages to stderr.log
	_ = syscall.Dup2(int(f.Fd()), 2)

	uiC := UIController{
		SongState: state,
		InputChan: uiInput,
	}

	uiC.launchUI()
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
