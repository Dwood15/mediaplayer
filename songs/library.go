package songs

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type (
	SongLibrary struct {
		Songs     []SongFile    `json:"songs,omitempty"`
		TotalTime time.Duration `json:"total_time,omitempty"`
		Pruned    bool          `json:"pruned,omitempty"`
		NextSong  int           `json:"next_song,omitempty"`
		lbWg      sync.WaitGroup
		LibInfo
	}

	LibInfo struct {
		AvgPlays    float64 `json:"avg_plays,omitempty"`
		AvgSkips    float64 `json:"avg_skips,omitempty"`
		AvgScore    float64 `json:"avg_score,omitempty"`
		LastCompute int64   `json:"last_compute_time,omitempty"`

		NumSkips   uint64        `json:"total_skips,omitempty"`
		NumPlays   uint64        `json:"total_plays,omitempty"`
		TotalScore float64       `json:"total_score,omitempty"`
		TimePlayed time.Duration `json:"total_time_played,omitempty"`
	}
)

var libDir = ""
var maxSize int
var mu sync.RWMutex

func SetPlaylistMaxSize(max int) {
	if max == 0 {
		max = 25
		fmt.Println("defaulting the max playlist selection to 25")
	}

	maxSize = max
}

func SetLibraryDir(dir string) {
	if len(dir) == 0 {
		panic("no dir provided to search for music")
	}

	libDir = dir
}

func (lib *SongLibrary) Play() {
	numSongs := len(lib.Songs)

	if numSongs == 0 {
		panic("can't play any songs without a library")
	}

	if maxSize > numSongs {
		maxSize = numSongs
	} else if maxSize == 0 {
		maxSize = int(math.Floor(0.01*float64(len(lib.Songs)))) + 1
	}

	if lib.NextSong >= maxSize {
		fmt.Println("end of playlist, time to calculate next song.")
		lib.computeScores()
		lib.computePlaylist()
	}

	lib.Songs[lib.NextSong].Play()
	lib.NextSong++
}

func (lib *SongLibrary) LoadFromFiles() {
	fmt.Println("loading songs from files")

	lib.lbWg.Add(1)
	getSongs(libDir)

	lib.lbWg.Wait()

	fmt.Println("songs loaded, pulling ")

}

func getSongs(dir string) {
	defer lib.lbWg.Done()

	//sleep the goroutine anywhere between 0 and 2 seconds :thonk:
	time.Sleep(time.Duration(rand.Int63n(int64(2 * time.Second))))

	f, err := os.OpenFile(dir, os.O_RDONLY, os.ModeDir)
	if err != nil {
		panic(err)
	}

	dirInfo, err := f.Readdir(0)
	if err != nil {
		panic(err)
	}

	songs := make([]SongFile, 0, len(dirInfo))

	for _, fInfo := range dirInfo {
		nam := fInfo.Name()

		if fInfo.IsDir() {
			lib.lbWg.Add(1)

			go func(d, n string) {
				getSongs(d + "/" + n)
			}(dir, nam)
			continue
		}

		if fInfo.Size() < 1024 {
			//prune useless dropbox attrs files.
			if strings.HasSuffix(nam, "com.dropbox.attributes") {
				_ = os.Remove(nam)
			}

			continue
		}

		if strings.HasSuffix(nam, ".mp3") {
			song := SongFile{FileName: dir + "/" + nam}
			if err := song.loadPlayTime(); err != nil {
				//fmt.Println("error loading mp3 file: ", err.Error())
				continue
			}

			if song.PlayTime < 1*time.Minute+29*time.Second {
				continue
			}

			songs = append(songs, song)
			continue
		}
	}

	if len(songs) > 0 {
		mu.Lock()
		lib.Songs = append(lib.Songs, songs...)
		mu.Unlock()
	}
}

func (lib *SongLibrary) prune() {
	songs := lib.Songs[:0]

	for _, song := range lib.Songs {
		err := song.loadPlayTime()

		if err == nil && song.PlayTime > 1*time.Minute+29*time.Second {
			songs = append(songs, song)
		}
	}

	mu.Lock()
	lib.Songs = songs
	mu.Unlock()

	lib.Pruned = true
}

func (lib *SongLibrary) computePlaylist() {
	fmt.Println("computing playlist now")

	if n := len(lib.Songs); maxSize > n {
		fmt.Println("max size reduced to match the number of songs")
		maxSize = n
	}

	lib.NextSong = 0
	fmt.Println("playlist computed")
}

type byScore []SongFile

func (b byScore) Len() int           { return len(b) }
func (b byScore) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byScore) Less(i, j int) bool { return b[i].Score < b[j].Score }

func (lib *SongLibrary) computeScores() {
	mu.Lock()
	lib.TotalTime = 0
	lib.NumPlays = 0
	lib.NumSkips = 0
	lib.TotalScore = 0

	//O(n) loop
	for i := 0; i < len(lib.Songs); i++ {
		if lib.Songs[i].PlayTime == 0 {
			panic(fmt.Errorf("when computing scores, a song: %s was fount to have no PlayTime", lib.Songs[i].FileName))
		}

		lib.NumPlays += lib.Songs[i].TotalPlays
		lib.TotalTime += lib.Songs[i].PlayTime
		lib.NumSkips += lib.Songs[i].TotalSkips

		lib.Songs[i].computeScore()
		// we only care about the scores of songs that are positive.
		if lib.Songs[i].Score > 0 {
			lib.TotalScore += lib.Songs[i].Score
		}
	}

	lib.AvgPlays = float64(lib.NumPlays) / float64(len(lib.Songs))
	lib.AvgSkips = float64(lib.NumSkips) / float64(len(lib.Songs))
	lib.AvgScore = lib.TotalScore / float64(len(lib.Songs))

	lib.LastCompute = time.Now().Unix()

	//O(n*log(n))
	sort.Sort(sort.Reverse(byScore(lib.Songs)))
	mu.Unlock()
	fmt.Println("scores computed and sorted")
}
