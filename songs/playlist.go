package songs

type (
	Playlist struct {
		nextSong    int
		SongsToPlay []int
	}
)

func (p *Playlist) NextSong() bool {
	if p.nextSong >= len(p.SongsToPlay)-1 {
		return true
	}

	n := p.SongsToPlay[p.nextSong]
	next := lib.Songs[n]

	next.Play()

	p.nextSong++
	return false
}
