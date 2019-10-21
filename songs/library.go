package songs

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type (
	LibInfo struct {
		AvgPlays    float64
		AvgSkips    float64
		LastCompute time.Time

		UniquePlaysSinceCompute uint64
		NumPlays                uint64
		TimePlayed              time.Duration
	}

	SongLibrary struct {
		Songs []SongFile
		LibInfo

		TotalTime time.Duration
		Pruned    bool
	}
)

func (lib *SongLibrary) LoadFromFiles() {
	const dir = "data/"
	const ext = ".mp3"

	f, err := os.OpenFile(dir, os.O_RDONLY, os.ModeDir)
	if err != nil {
		panic(err)
	}

	infos, err := f.Readdir(0)
	if err != nil {
		panic(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

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
			lib.Songs = append(lib.Songs, SongFile{FileName: wd + n})
		}
	}

}

func (lib *SongLibrary) prune() {
	songs := lib.Songs[:0]

	for _, song := range lib.Songs {
		if song.PlayTime > 1*time.Minute+29*time.Second {
			songs = append(songs, song)
		}
	}

	lib.Songs = songs
	lib.Pruned = true
}

func (lib *SongLibrary) computeScores() {
	for _, song := range lib.Songs {
		song.computeScore(lib.LibInfo)
	}

	lib.UniquePlaysSinceCompute = 0
	lib.LastCompute = time.Now()
}
