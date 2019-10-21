package songs

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type (
	LibInfo struct {
		AvgPlays    float64
		AvgSkips    float64
		AvgScore    float64
		LastCompute time.Time

		NumSkips   uint64
		NumPlays   uint64
		TotalScore float64
		TimePlayed time.Duration
	}

	SongLibrary struct {
		Songs []SongFile
		LibInfo
		TotalTime time.Duration
		Pruned    bool
		ToPlay    Playlist
	}
)

var mu sync.RWMutex

func getSongs(infos []os.FileInfo) []SongFile {
	const ext = ".mp3"
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	wd += "/data/"
	var songs []SongFile
	for _, fInfo := range infos {
		n := fInfo.Name()
		if fInfo.IsDir() {
			fmt.Println("found dir: " + n)
			continue
		}

		if fInfo.Size() < 1024 {
			continue
		}

		if strings.HasSuffix(n, ext) {
			fmt.Println("found mp3 file: " + n)

			song := SongFile{FileName: wd + n}
			song.loadPlayTime()
			songs = append(songs, song)
		}
	}

	return songs
}

func (lib *SongLibrary) LoadFromFiles() {
	const dir = "data/"

	f, err := os.OpenFile(dir, os.O_RDONLY, os.ModeDir)
	if err != nil {
		panic(err)
	}

	infos, err := f.Readdir(0)
	if err != nil {
		panic(err)
	}

	songs := getSongs(infos)

	//don't lock until we actually get to.
	mu.Lock()
	lib.Songs = songs
	mu.Unlock()

}

func (lib *SongLibrary) prune() {
	songs := lib.Songs[:0]

	for _, song := range lib.Songs {
		if song.PlayTime == 0 {
			song.loadPlayTime()
		}

		if song.PlayTime > 1*time.Minute+29*time.Second {
			songs = append(songs, song)
		}
	}

	mu.Lock()
	lib.Songs = songs
	mu.Unlock()

	lib.Pruned = true
}

func (lib *SongLibrary) computeScores() {
	mu.Lock()
	lib.TotalTime = 0
	lib.NumPlays = 0
	lib.NumSkips = 0

	for _, song := range lib.Songs {
		if song.PlayTime == 0 {
			song.loadPlayTime()
		}

		lib.NumPlays += song.TotalPlays
		lib.TotalTime += song.PlayTime
		lib.NumSkips += song.TotalSkips
	}

	lib.AvgPlays = float64(lib.NumPlays) / float64(len(lib.Songs))
	lib.AvgSkips = float64(lib.NumSkips) / float64(len(lib.Songs))

	lib.TotalScore = 0
	for _, song := range lib.Songs {
		song.computeScore()
		lib.TotalScore += song.Score
	}
	lib.AvgScore = lib.TotalScore / float64(len(lib.Songs))

	lib.LastCompute = time.Now()

	sort.Sort(sort.Reverse(ByScore(lib.Songs)))
	mu.Unlock()
	fmt.Println("scores computed and sorted")
}

func (lib *SongLibrary) computePlaylist() {
	fmt.Println("computing playlist now")

	plSize := int(math.Floor(0.1*float64(len(lib.Songs)))) + 1

	lib.ToPlay.SongsToPlay = lib.Songs[:plSize]

	lib.ToPlay.nextSong = 0
	fmt.Println("playlist computed")
}

func (lib *SongLibrary) Play() {
	if lib.ToPlay.NextSong() {
		lib.computeScores()
		lib.computePlaylist()
	}
}
