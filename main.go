package main

import (
	"fmt"
	"github.com/dwood15/mediaplayer/songplayer"
	"golang.org/x/sys/unix"
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
	songplayer.SetLibraryDir(cfg.MusicDir)
	songplayer.SetPlaylistMaxSize(cfg.MaxPlaylistSize)
	go handleShutdown()
}

func socTest() {
	fd, sAU := songplayer.InitSock()

	if fd < 0 || sAU == nil {
		panic("initsock fail")
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var nfd int

	go func() {
		defer wg.Done()
		nfd = songplayer.ListenForConection(fd)

		if nfd < 0 {
			wg.Done()
			panic("server listen for connection")
		}
	}()

	go func() {
		defer wg.Done()
		//time.Sleep(15 * time.Millisecond)

		cfd := songplayer.OpenClientfd()

		if cfd < 0 {
			wg.Done() //force the server to be done
			panic("client connection fails")
		}

		n, err := unix.Write(cfd, []byte("some data"))

		if err != nil {
			wg.Done() //force the server to be done
			panic("writing to sock: " + unix.ErrnoName(err.(unix.Errno)))
		}

		if n == 0 {
			wg.Done() //force the server to be done
			panic("no bytes written")
		}
	}()

	wg.Wait()

	toread := make([]byte, 100)

	n, _, err := unix.Recvfrom(nfd, toread, unix.MSG_DONTWAIT)

	if err != nil {
		panic("init sock srv read " + err.Error())
	}

	if n == 0 {
		panic("no bytes rcvd from client")
	}

	fmt.Printf("read: %s\n", string(toread))
}


func main() {
	socTest()

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
