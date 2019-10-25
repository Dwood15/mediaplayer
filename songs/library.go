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
	//SongLibrary is a large structure for containing the state of the media player
	SongLibrary struct {
		Songs     []SongFile    `json:"songs,omitempty"`
		TotalTime time.Duration `json:"total_time,omitempty"`
		Pruned    bool          `json:"pruned,omitempty"`
		NextSong  int           `json:"next_song,omitempty"`
		lbWg      sync.WaitGroup
		mu        sync.RWMutex
		LibInfo
	}

	//LibInfo Provides metadata and basic statistics around the player state
	LibInfo struct {
		AvgPlays    float64 `json:"avg_plays,omitempty"`
		AvgSkips    float64 `json:"avg_skips,omitempty"`
		AvgScore    uint64  `json:"avg_score,omitempty"`
		LastCompute int64   `json:"last_compute_time,omitempty"`

		NumSkips   uint64        `json:"total_skips,omitempty"`
		NumPlays   uint64        `json:"total_plays,omitempty"`
		TotalScore uint64        `json:"total_score,omitempty"`
		TimePlayed time.Duration `json:"total_time_played,omitempty"`
	}
)

var libDir = ""
var maxSize = 25

//SetPlaylistMaxSize indicates to the player at what interval of played songs it should initiate computes.
// if unspecified, the maxSize defaults to 25 songs
func SetPlaylistMaxSize(max int) {
	if max == 0 {
		max = 25
	}

	maxSize = max
}

//SetLibraryDir sets the library to the specified directory folder
func SetLibraryDir(dir string) {
	if len(dir) == 0 {
		panic("no dir provided to search for music")
	}

	libDir = dir
}

func (lib *SongLibrary) NextSongFiles(num int) (s []SongFile) {
	if num <= 0 {
		return nil
	}

	return append(s, lib.Songs[lib.NextSong-1:lib.NextSong+num-1]...)
}

//Play begins the cycle of playing songs
func (lib *SongLibrary) BeginPlaying() {
	numSongs := len(lib.Songs)

	if numSongs == 0 {
		panic("can't play any songs without a library")
	}

	if maxSize > numSongs {
		maxSize = numSongs
	} else if maxSize == 0 {
		maxSize = int(math.Floor(0.01*float64(len(lib.Songs)))) + 1
	}

	for !lib.Songs[lib.NextSong].play() {
		if lib.NextSong >= maxSize {
			lib.computeScores()
			lib.computePlaylist()
		}

		lib.NextSong++
		lib.persistSelf()
	}
}

//LoadFromFiles initiates recursive directory scanning to find mp3 files.
func (lib *SongLibrary) LoadFromFiles() {
	lib.lbWg.Add(1)
	getSongs(libDir)

	lib.lbWg.Wait()
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
		lib.mu.Lock()
		lib.Songs = append(lib.Songs, songs...)
		lib.mu.Unlock()
	}
}

func (lib *SongLibrary) computePlaylist() {
	if n := len(lib.Songs); maxSize > n {
		maxSize = n
	}

	lib.NextSong = 0
}

//Utilities for sorting the library of songs
type byScore []SongFile

func (b byScore) Len() int           { return len(b) }
func (b byScore) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byScore) Less(i, j int) bool { return b[i].Score < b[j].Score }

func (lib *SongLibrary) simpleCompute() {
	lib.mu.Lock()
	for i := 0; i < lib.NextSong; i++ {
		lib.Songs[i].computeScore()
	}
	sort.Sort(sort.Reverse(byScore(lib.Songs)))
	lib.NextSong = 0
	lib.mu.Unlock()
}

func (lib *SongLibrary) computeScores() {
	lib.mu.Lock()
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
	lib.AvgScore = lib.TotalScore / uint64(len(lib.Songs))

	lib.LastCompute = time.Now().Unix()
	lib.NextSong = 0
	//O(n*log(n))
	sort.Sort(sort.Reverse(byScore(lib.Songs)))
	lib.mu.Unlock()
}

//Currently unused function, explicitly for
func (lib *SongLibrary) prune() {
	songs := lib.Songs[:0]

	for _, song := range lib.Songs {
		err := song.loadPlayTime()

		if err == nil && song.PlayTime > 1*time.Minute+29*time.Second {
			songs = append(songs, song)
		}
	}

	lib.mu.Lock()
	lib.Songs = songs
	lib.Pruned = true
	lib.mu.Unlock()

}
