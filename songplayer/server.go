package songplayer

import (
	"fmt"
	"golang.org/x/sys/unix"
	"syscall"
)

const sockName = "/tmp/mediaplayer.sock"

func InitSock() int {
	//ok if this fails - we're just clearing out the sock if it still exists.
	_ = unix.Unlink(sockName)

	fd, err := unix.Socket(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_NONBLOCK, 0)

	if err != nil {
		fmt.Println("main failed to initialize socket")
		panic(err)
	}

	addr := &unix.SockaddrUnix{
		Name: sockName,
	}

	if err = unix.Bind(fd, addr); err != nil {
		panic(err)
	}

	var ok bool
	var sAU *unix.SockaddrUnix
	var flg int

	sN, err := unix.Getsockname(fd)

	if err != nil {
		fmt.Println("failed getting sockname: ", err.Error())
		goto errOut
	}

	sAU, ok = sN.(*unix.SockaddrUnix)

	if !ok {
		fmt.Println("failed with sock addr unix cast")
		goto errOut
	}

	fmt.Println("got sockName: ", sAU.Name)

	//user-only rwx perms
	if err = unix.Chmod(sockName, unix.S_IRWXU); err != nil {
		goto errOut
	}

	fmt.Println("unix socket addr: ", addr.Name, " getsockname: ", sAU.Name)

	//Tells the OS we're ready to listen on the socket.
	if err = unix.Listen(fd, 10); err != nil {
		fmt.Println("failed connecting: ", err.Error())
		goto errOut
	}

	//Unnecessary checks for ensuring file descriptor state
	if flg, err = unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0); err != nil {
		fmt.Println("FcntlInt: ", err.Error())
		goto errOut
	}

	if flg&unix.SOCK_NONBLOCK == 0 {
		fmt.Println("checking for blocking socket - unblocked sock")
		goto errOut
	}

	fmt.Println("flag found: ", flg)

	return fd

errOut:
	syscall.Close(fd)
	_ = syscall.Unlink(sockName)
	return -1
}

func ListenForConection(fd int) int {
	var ok bool
	var sAU *unix.SockaddrUnix

unixAccept:
	nfd, sa, err := unix.Accept(fd)

	if err != nil {
		if !(err.(unix.Errno)).Temporary() {
			goto errOut
		}

		goto unixAccept
	}
	goto done

done:
	if err != nil {

		fmt.Println("LFC socket accept: ", err.Error())
		goto errOut
	}

	sAU, ok = sa.(*unix.SockaddrUnix)

	if !ok {
		fmt.Println("LFC connection, cast to unix socket")
		goto errOut
	}

	fmt.Println("LFC socket unix address: ", sAU.Name)

	return nfd

errOut:
	syscall.Close(fd)
	//_ = syscall.Unlink(sAU.Name)
	return -1
}

func LaunchServer() {

}
