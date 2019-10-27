package songplayer

import (
	"sync"
	"testing"
)

func TestInitSock(t *testing.T) {
	t.Parallel()

	fd := InitSock()

	if fd < 0 {
		t.Fatal()
	}

	t.Logf("fd passed, id: %d", fd)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		nfd := ListenForConection(fd)

		if nfd < 0 {
			t.Fatal()
		}
		t.Logf("nfd passed, id: %d", nfd)
		wg.Done()
	}()





	wg.Wait()
}
