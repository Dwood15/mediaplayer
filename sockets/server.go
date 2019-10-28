// +build linux

package sockets

import (
	"fmt"
	"golang.org/x/sys/unix"
	"syscall"
	"time"
)

type Server struct {
	SockName        string
	OnConnect       func(int, chan bool)
	shouldClose     bool
	openConnections []chan bool
}

//LaunchServer opens a socket and listens for connections, calling onconnect as a goroutine
//when it notices a connection
func (s *Server) LaunchServer() error {
	//ok if this fails - we're just clearing out the sock if it still exists.
	_ = unix.Unlink(s.SockName)

	fd, err := unix.Socket(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_NONBLOCK, 0)

	if err != nil {
		fmt.Println("main failed to initialize socket")
		return err
	}

	if err = unix.Bind(fd, &unix.SockaddrUnix{Name: s.SockName}); err != nil {
		return err
	}

	//ensure user-only rwx perms
	if err = unix.Chmod(s.SockName, unix.S_IRWXU); err != nil {
		goto errOut
	}

	//Tell the OS we're ready to listen on the socket.
	if err = unix.Listen(fd, 10); err != nil {
		fmt.Println("failed connecting: ", err.Error())
		goto errOut
	}

	{
		var flg int

		//Possibly-unnecessary checks for ensuring file descriptor state
		if flg, err = unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0); err != nil {
			fmt.Println("FcntlInt: ", err.Error())
			goto errOut
		}

		if flg&unix.SOCK_NONBLOCK == 0 {
			fmt.Println("checking for blocking socket - unblocked sock")
			goto errOut
		}

		fmt.Println("flag found: ", flg)
	}

	go s.listenForConections(fd)
	return nil

errOut:
	syscall.Close(fd)
	_ = syscall.Unlink(s.SockName)
	return err
}

func (s *Server) CloseServer() {
	s.shouldClose = true
}

func (s *Server) listenForConections(fd int) {
	defer syscall.Close(fd)

	onConnClose := make(chan bool)

	var numOpen int

	for waitFor := 1 * time.Millisecond; s.shouldClose == false; <-time.After(waitFor) {
		nfd, _, err := unix.Accept(fd)

		if err == nil {
			conn := make(chan bool)
			numOpen++
			go func() {
				defer unix.Close(nfd)
				s.OnConnect(nfd, conn)
				onConnClose <- true
			}()
			s.openConnections = append(s.openConnections, conn)
			continue
		}

		if !(err.(unix.Errno)).Temporary() {
			panic("non-temporary error received")
		}

		if numOpen > 0 && waitFor < 2*time.Second {
			waitFor += 3 * time.Millisecond
		}

		select {
		case <-onConnClose:
			numOpen--
		}
	}
}
