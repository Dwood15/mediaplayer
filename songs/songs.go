package songs

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"math"
	"math/rand"
	"os"
	"time"
)

type (
	PlayInfo struct {
		Score             float64   `json:"score,omitempty"`
		TotalSkips        uint64    `json:"total_skips,omitempty"`
		ConsecutiveSkips  float64   `json:"consecutive_skips,omitempty"`
		LastSkipped       time.Time `json:"last_skipped_time,omitempty"`
		LastPlayed        time.Time `json:"last_played_time,omitempty"`
		TotalPlays        uint64    `json:"total_plays,omitempty"`
		ComputesSincePlay uint8     `json:"computes_since_last_play,omitempty"`
	}
	SongFile struct {
		FileName string        `json:"file_name,omitempty"`
		PlayTime time.Duration `json:"play_time,omitempty"`
		PlayInfo
	}
)

func (sF *SongFile) loadPlayTime() error {
	f, err := os.Open(sF.FileName)
	if err != nil {
		panic(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return err
	}
	defer streamer.Close()

	sF.PlayTime = format.SampleRate.D(streamer.Len())

	return nil
}

func (sF *SongFile) Play() {
	f, err := os.Open(sF.FileName)
	if err != nil {
		panic(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		panic(err)
	}
	defer streamer.Close()
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/2))

	playStart := time.Now()
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	for {
		select {
		case <-done:
			mu.Lock()
			sF.TotalPlays++
			lib.NumPlays++
			lib.TimePlayed += time.Since(playStart)
			mu.Unlock()
			return
			//case <-time.After(time.Second):
			//	speaker.Lock()
			//	fmt.Println(format.SampleRate.D(streamer.Position()).Round(time.Second))
			//	speaker.Unlock()
		}
	}
}

func (pI *PlayInfo) computeSkipScore() bool {
	//Compute the lastSkipped scores
	if pI.LastSkipped.After(lib.LastCompute) {
		pI.Score -= 15 * (1 + pI.ConsecutiveSkips)

		if pI.TotalSkips > uint64(math.Floor(lib.AvgSkips)) {
			pI.Score -= 15
		}

		pI.ConsecutiveSkips++

		return false
	}

	if pI.LastSkipped.Before(pI.LastPlayed) {
		pI.ConsecutiveSkips = 0
	}

	//give the skipped songs a bit of attrition
	pI.Score += 5
	return true
}

func (pI *PlayInfo) computePlayScore() {
	if pI.LastPlayed.Before(lib.LastCompute) {
		pI.ComputesSincePlay++
		pI.Score += 15 * float64(pI.ComputesSincePlay)
	}

	//We've just played the song, so we're going to drop its score somewhat.
	if pI.LastPlayed.After(lib.LastCompute) && pI.Score > lib.AvgScore {
		pI.Score -= (pI.Score - lib.AvgScore) / 2
	}
}

func (pI *PlayInfo) computeScore() {
	//give new songs some extra jitter.
	if pI.Score == 0 {
		//[0, numSongs]
		pI.Score += math.Floor(float64(len(lib.Songs)) * rand.Float64())
	}

	//[0, 5]
	pI.Score += math.Floor(5 * rand.Float64())

	if !pI.computeSkipScore() {
		return
	}

	pI.computePlayScore()
}
