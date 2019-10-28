package songplayer

import (
	"fmt"
	"golang.org/x/sys/unix"
)

func OpenClientfd() int {
	fd, err := unix.Socket(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_NONBLOCK, 0)

	if err != nil {
		fmt.Println("client initialize socket: ", err)
		panic(err)
	}

	addr := &unix.SockaddrUnix{
		Name: sockName,
	}

	if err := unix.Connect(fd, addr); err != nil {
		fmt.Println("client connecting to socket: ", unix.ErrnoName(err.(unix.Errno)))
		_ = unix.Close(fd)
		return -1
	}

	return fd
}