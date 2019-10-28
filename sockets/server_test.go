// +build linux

package sockets

import (
	"golang.org/x/sys/unix"
	"sync"
	"testing"
)

const testSock = "/tmp/gomediaplayer_test.sock"

func TestInitSock(t *testing.T) {
	t.Parallel()

	srv := Server{
		SockName: testSock,
		OnConnect: func(cFD int, done chan bool) {
			toread := make([]byte, 100)

			n, _, err := unix.Recvfrom(cFD, toread, unix.MSG_DONTWAIT)

			if err != nil {
				t.Fatal("init sock srv read ", err)
			}

			if n == 0 {
				t.Fatal("no bytes rcvd from client")
			}

			t.Logf("read: %s\n", string(toread))
		},
	}

	srv.LaunchServer()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		//time.Sleep(15 * time.Millisecond)

		cfd := OpenClientfd(testSock)

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

	t.Logf("nfd passed")

}
