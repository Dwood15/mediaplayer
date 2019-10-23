package songs

import (
	"fmt"
	"math"
)

type (
	Playlist struct {
		nextSong    int
		SongsToPlay []SongFile
		maxSize     int
	}
)

func (p *Playlist) NextSong() bool {
	numSongs := len(p.SongsToPlay)

	if p.maxSize > numSongs {
		p.maxSize = numSongs
	} else if p.maxSize == 0 {
		p.maxSize = int(math.Floor(0.01*float64(len(lib.Songs)))) + 1
	}

	if p.nextSong >= p.maxSize {
		fmt.Println("end of playlist, time to calculate next song.")
		return true
	}

	p.SongsToPlay[p.nextSong].Play()

	p.nextSong++
	return false
}
