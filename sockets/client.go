// +build linux

package sockets

import (
	"encoding/binary"
	"fmt"
	"sort"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/dwood15/mediaplayer/songplayer"
)

//Client
type Client struct {
	Addr *unix.SockaddrUnix
}

type fielded struct {
	offset int
	t      int
}

const ssIdx = int(unsafe.Offsetof(songplayer.PlayingSong{}.SongScore))
const slIdx = int(unsafe.Offsetof(songplayer.PlayingSong{}.SongLength))
const sNIdx = int(unsafe.Offsetof(songplayer.PlayingSong{}.CurrentSong))
const szOf = int(unsafe.Sizeof(songplayer.PlayingSong{}))

var (
	//as-of-yet-unused helpers for decoding structures
	sortedIdxs = [3]fielded{{offset: ssIdx, t: 0}, {offset: slIdx, t: 1}, {offset: sNIdx, t: 2}}

	idxOfSS int
	idxOfSL int
	idxOfSN int
)

type byScore [3]fielded

func (b byScore) Len() int           { return len(b) }
func (b byScore) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byScore) Less(i, j int) bool { return b[i].offset < b[j].offset }

func init() {
	sort.Sort(byScore(sortedIdxs))

	for i, v := range sortedIdxs {
		switch v.t {
		case 0:
			idxOfSS = i
		case 1:
			idxOfSL = i
		case 2:
			idxOfSN = i
		}
	}
}

func handleRcv(fd int, rcvd []byte, onSongUpdate chan songplayer.PlayingSong) {
	n, err := unix.Read(fd, rcvd)
	if err != nil && !err.(unix.Errno).Temporary() {
		panic("launch client: " + err.Error())
	} else if n == -1 {
		time.Sleep(1 * time.Millisecond)
		return
	}

	fmt.Println("bytes Rcv'd: ", string(rcvd))
	fmt.Printf("copied bytes [%d] vs: [%d] (size of struct)\n", n, szOf)

	ss, _ := binary.Uvarint(rcvd[ssIdx : ssIdx+10 : szOf])
	sl, _ := binary.Varint(rcvd[slIdx : slIdx+10])
	sp := songplayer.PlayingSong{
		SongScore:   ss,
		SongLength:  time.Duration(sl),
		CurrentSong: string(rcvd[sNIdx:]),
	}

	fmt.Printf("what I think the string would look like: [%s]", sp.CurrentSong)
	onSongUpdate <- sp
}

//LaunchClient takes sockname
func (c *Client) LaunchClient(onInput chan int64, onSongUpdate chan songplayer.PlayingSong) error {
	fd, err := unix.Socket(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_NONBLOCK, 0)

	if err != nil {
		return err
	}

	if err := unix.Connect(fd, c.Addr); err != nil {
		fmt.Println("unix connect err")
		_ = unix.Close(fd)
		return err
	}

	go func() {
		var toSend int64
		sendBuf := make([]byte, 10)
		rcvd := make([]byte, unsafe.Sizeof(songplayer.PlayingSong{})+32)

		fmt.Println("Client now handling the recv loop")

		for {
			handleRcv(fd, rcvd, onSongUpdate)

			select {
			case toSend = <-onInput:
				fmt.Println("found data to send")
				binary.PutVarint(sendBuf, toSend)
			}

			if toSend != 0 {
			trySend:
				fmt.Println("found data to send")
				if _, err := unix.Write(fd, sendBuf); err != nil {
					if err.(unix.Errno).Temporary() {
						fmt.Println("temp to send error, trying to recv first")

						handleRcv(fd, rcvd, onSongUpdate)
						fmt.Println("handleRcv already happened, trying again")
						time.Sleep(1 * time.Millisecond)
						goto trySend
					} else {
						fmt.Println("non-temporary error trying to send")
						panic(err)
					}
				}
				toSend = 0
			}
		}
	}()

	return nil
}
