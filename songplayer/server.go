package songplayer

import (
	"fmt"
	"golang.org/x/sys/unix"
	"syscall"
)

func initSock() {
	//ok if this fails - we're just clearing out the sock if it still exists.
	_ = syscall.Unlink("/tmp/mediaplayer.sock")

	fd, err := unix.Socket(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_NONBLOCK, 0)

	defer unix.Close(fd)

	if err != nil {
		fmt.Println("main failed to initialize socket")
		panic(err)
	}

	addr := &unix.SockaddrUnix{
		Name: "mediaplayer.sock",
	}

	if err := unix.Bind(fd, addr); err != nil {
		panic(err)
	}

	sN, _ := syscall.Getsockname(fd)

	fmt.Println("unix socket addr: ", addr.Name, " getsockname: ", sN)
}

func LaunchServer() {

}
