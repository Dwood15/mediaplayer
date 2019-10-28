// +build linux

package sockets

import (
	"encoding/binary"
	"fmt"
	"github.com/dwood15/mediaplayer/songplayer"
	"golang.org/x/sys/unix"
	"sort"
	"time"
	"unsafe"
)

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
	sortedIdxs = [3]fielded{{offset: ssIdx, t: 0}, {offset: slIdx, t: 1}, {offset: sNIdx, t: 2}}
	idxOfSS    int
	idxOfSL    int
	idxOfSN    int
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

//LaunchClient takes sockname
func (c *Client) LaunchClient(onInput chan int64, onSongUpdate chan songplayer.PlayingSong) error {
	fd, err := unix.Socket(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_NONBLOCK, 0)

	if err != nil {
		return err
	}

	if err := unix.Connect(fd, c.Addr); err != nil {
		_ = unix.Close(fd)
		return err
	}

	var toSend int64
	sendBuf := make([]byte, 10)
	rcvd := make([]byte, unsafe.Sizeof(songplayer.PlayingSong{}))

	handleRcv := func() {
		n, _, err := unix.Recvfrom(fd, rcvd, 0)
		if err != nil && !err.(unix.Errno).Temporary() {
			panic("launch client: " + err.Error())
		} else if n == 0 {
			return
		}

		fmt.Println("bytes Rcv'd: ", string(rcvd))

		if n < szOf {
			fmt.Printf("wow, copied byted [%d] != [%d] (size of struct):\n", n, szOf)
		}

		ss, _ := binary.Uvarint(rcvd[ssIdx : ssIdx+10 : szOf])
		sl, _ := binary.Varint(rcvd[slIdx:10])
		sp := songplayer.PlayingSong{
			SongScore:  ss,
			SongLength: time.Duration(sl),
		}

		fmt.Println("printing what I think the string would look like")
		fmt.Println(string(rcvd[sNIdx:szOf]))
		onSongUpdate <- sp
	}

	for {
		select {
		case toSend = <-onInput:
			binary.PutVarint(sendBuf, toSend)
		case <-time.After(2 * time.Millisecond):
		}

		handleRcv()

		if toSend != 0 {
			if err := unix.Sendto(fd, sendBuf, 0, c.Addr); err != nil {

			}
			toSend = 0
		}

		handleRcv()

	}
}
