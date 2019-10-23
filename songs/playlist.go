package songs

import (
	"fmt"
	"math"
)

type (
	Playlist struct {
		nextSong    int
		SongsToPlay []SongFile
	}
)

func (p *Playlist) NextSong() bool {
	numSongs := len(p.SongsToPlay)

	if numSongs == 0 {
		panic("can't play songs with empty playlist")
	}

	if maxSize > numSongs {
		maxSize = numSongs
	} else if maxSize == 0 {
		maxSize = int(math.Floor(0.01*float64(len(lib.Songs)))) + 1
	}

	if p.nextSong >= maxSize {
		fmt.Println("end of playlist, time to calculate next song.")
		return true
	}

	p.SongsToPlay[p.nextSong].Play()

	p.nextSong++
	return false
}
