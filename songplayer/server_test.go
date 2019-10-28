// +build linux

package songplayer

import (
	"golang.org/x/sys/unix"
	"sync"
	"testing"
)

func TestInitSock(t *testing.T) {
	t.Parallel()
	fd, sAU := InitSock()

	if fd < 0 || sAU == nil {
		t.Fatal()
	}

	t.Logf("fd passed, id: %d", fd)

	var wg sync.WaitGroup
	wg.Add(2)

	var nfd int

	go func() {
		defer wg.Done()
		nfd = ListenForConection(fd)

		if nfd < 0 {
			wg.Done()
			t.Fatal("server listen for connection")
		}
	}()

	go func() {
		defer wg.Done()
		//time.Sleep(15 * time.Millisecond)

		cfd := OpenClientfd()

		if cfd < 0 {
			wg.Done() //force the server to be done
			t.Fatal("client connection")
		}

		n, err := unix.Write(cfd, []byte("some data"))

		if err != nil {
			wg.Done() //force the server to be done
			t.Fatal("writing to sock: ", unix.ErrnoName(err.(unix.Errno)))
		}

		if n == 0 {
			wg.Done() //force the server to be done
			t.Fatal("no bytes written")
		}
	}()

	wg.Wait()

	t.Logf("nfd passed, id: %d", nfd)

	toread := make([]byte, 100)

	n, _, err := unix.Recvfrom(nfd, toread, unix.MSG_DONTWAIT)

	if err != nil {
		t.Fatal("init sock srv read ", err)
	}

	if n == 0 {
		t.Fatal("no bytes rcvd from client")
	}

	t.Logf("read: %s\n", string(toread))
}
