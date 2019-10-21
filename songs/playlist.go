package songs

import "fmt"

type (
	Playlist struct {
		nextSong    int
		SongsToPlay []SongFile
	}

	ByScore []SongFile
)

func (b ByScore) Len() int           { return len(b) }
func (b ByScore) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByScore) Less(i, j int) bool { return b[i].Score < b[j].Score }

func (p *Playlist) NextSong() bool {
	numSongs := len(p.SongsToPlay)
	fmt.Printf("nextSong begins. num to play: %v ", numSongs)

	
	if p.nextSong >= numSongs {
		fmt.Println("end of playlist, time to calculate next song.")
		return true
	}

	p.SongsToPlay[p.nextSong].Play()

	p.nextSong++
	return false
}
