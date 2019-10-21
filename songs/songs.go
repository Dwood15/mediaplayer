package songs

import (
	"fmt"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"os"
	"time"
)

type (
	PlayInfo struct {
		Score             float64
		TotalSkips        float64
		ConsecutiveSkips  float64
		LastSkipped       time.Time
		LastPlayed        time.Time
		TotalPlays        float64
		ComputesSincePlay float64
	}
	SongFile struct {
		FileName string
		PlayInfo
		PlayTime time.Duration
		streamer *beep.Streamer
	}
)

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

	done := make(chan bool)

	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	for {
		select {
		case <-done:
			return
		case <-time.After(time.Second):
			speaker.Lock()
			fmt.Println(format.SampleRate.D(streamer.Position()).Round(time.Second))
			speaker.Unlock()
		}
	}

}

func (pI *PlayInfo) computeSkipScore(lib LibInfo) bool {
	//Compute the lastSkipped scores
	if pI.LastSkipped.After(lib.LastCompute) {
		pI.Score -= 15 * (1 + pI.ConsecutiveSkips)

		if pI.TotalSkips > lib.AvgSkips {
			pI.Score -= 15
		}

		return false
	}

	if pI.LastSkipped.Before(pI.LastPlayed) {
		pI.ConsecutiveSkips = 0
	}

	pI.Score += 5
	return true
}

func (pI *PlayInfo) computePlayScore(lib LibInfo) {
	if pI.LastPlayed.Before(lib.LastCompute) {
		pI.ComputesSincePlay += 1
	}

	pI.Score += 15 * (1 + pI.ComputesSincePlay)

	if pI.TotalPlays < lib.AvgPlays {
		pI.Score += 15
	}
}

func (pI *PlayInfo) computeScore(lib LibInfo) {
	if !pI.computeSkipScore(lib) {
		return
	}

	pI.computePlayScore(lib)
}
