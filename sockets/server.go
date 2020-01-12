// +build linux

package sockets

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

type Server struct {
	SockAddr        *unix.SockaddrUnix
	OnConnect       func(int, chan bool)
	shouldClose     bool
	openConnections []chan bool
}

//LaunchServer opens a socket and listens for connections, calling onconnect as a goroutine
//when it notices a connection
func (s *Server) LaunchServer() error {
	//ok if this fails - we're just clearing out the sock if it still exists.
	_ = unix.Unlink(s.SockAddr.Name)

	fd, err := unix.Socket(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_NONBLOCK, 0)

	if err != nil {
		fmt.Println("main failed to initialize socket")
		return err
	}

	if err = unix.Bind(fd, s.SockAddr); err != nil {
		return err
	}

	//ensure user-only rwx perms
	if err = unix.Chmod(s.SockAddr.Name, unix.S_IRWXU); err != nil {
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
	}

	//fmt.Println("launching listenForConnections goroutine")
	go s.listenForConections(fd)
	return nil

errOut:
	syscall.Close(fd)
	_ = syscall.Unlink(s.SockAddr.Name)
	return err
}

func (s *Server) CloseServer() {
	s.shouldClose = true
}

func (s *Server) listenForConections(fd int) {
	defer syscall.Close(fd)

	onConnClose := make(chan bool)

	var numOpen int

	fmt.Println("server is listening for connections")
	for /*waitFor := 1 * time.Millisecond*/ s.shouldClose == false /*<-time.After(waitFor)*/ {
		nfd, _, err := unix.Accept(fd)

		if err == nil {
			conn := make(chan bool)
			numOpen++
			go func() {
				defer unix.Close(nfd)
				fmt.Println("launching client connection")
				s.OnConnect(nfd, conn)
				onConnClose <- true
			}()
			s.openConnections = append(s.openConnections, conn)
			fmt.Println("connection found and added to the open slice")
			continue
		}

		if !(err.(unix.Errno)).Temporary() {
			panic("non-temporary error received")
		}

		//if numOpen > 0 && waitFor < 1*time.Second {
		//	waitFor += 1 * time.Microsecond
		//}

		//select {
		//case <-onConnClose:
		//	fmt.Println("connection close detected")
		//	numOpen--
		//}
	}
	fmt.Println("closing server listenForConnections loop")
}
