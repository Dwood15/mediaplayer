package songs

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/gdamore/tcell"
	"math"
	"math/rand"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"
)

type (
	PlayInfo struct {
		Score             float64 `json:"score,omitempty"`
		TotalSkips        uint64  `json:"total_skips,omitempty"`
		ConsecutiveSkips  float64 `json:"consecutive_skips,omitempty"`
		LastSkipped       int64   `json:"last_skipped_time,omitempty"`
		LastPlayed        int64   `json:"last_played_time,omitempty"`
		TotalPlays        uint64  `json:"total_plays,omitempty"`
		ComputesSincePlay uint8   `json:"computes_since_last_play,omitempty"`
	}
	SongFile struct {
		FileName string        `json:"file_name,omitempty"`
		PlayTime time.Duration `json:"play_time,omitempty"`
		PlayInfo
	}
	PlayingSong struct {
		CurrentSong string
		SongScore   float64
		SongLength  time.Duration
	}
)

var SongState = make(chan PlayingSong)
var SongTime = make(chan time.Duration)
var HotkeyEvent = make(chan *tcell.EventKey)

var playMu sync.Mutex

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

func (sF *SongFile) updateSkipped(s time.Time) {
	sF.ConsecutiveSkips++
	sF.TotalSkips++
	sF.ComputesSincePlay = 0
	sF.LastSkipped = time.Now().Unix()
	lib.NumSkips++
	lib.TimePlayed += time.Since(s)
}

func (sF *SongFile) updatePlayed(s time.Time) {
	sF.TotalPlays++
	sF.LastPlayed = time.Now().Unix()
	sF.ConsecutiveSkips = 0
	sF.ComputesSincePlay = 0
	lib.NumPlays++
	lib.TimePlayed += time.Since(s)
}

func (sF *SongFile) Play() {
	playMu.Lock()
	defer playMu.Unlock()

	//Load the song file
	f, err := os.Open(sF.FileName)
	if err != nil {
		panic(err)
	}

	//load beep's StreamSeeker
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		panic(err)
	}
	defer streamer.Close()

	//Rather than muck about with funky math for different song sample rates,
	//let's just initialize it to the exact format every time - we're only gonna
	//be playing one song at a time, anyway.
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/2))
	if err != nil {
		panic(err)
	}

	//Concurrency-safe containers for goroutine crosstalk.
	var skipped atomic.Value
	var playStart atomic.Value
	var timePaused atomic.Value
	done := make(chan bool)

	//So we know when the stream completes
	seq := beep.Seq(streamer, beep.Callback(func() {
		done <- true
		SongTime <- 0
	}))

	//So we can pause songs
	ctrl := &beep.Ctrl{
		Paused:   false,
		Streamer: seq,
	}

	//So we know when the song started.
	playStart.Store(time.Now())
	speaker.Play(ctrl)

	//Signal to the ui what's playing. Perhaps an atomic.Value would be better?
	SongState <- PlayingSong{
		CurrentSong: path.Base(sF.FileName),
		SongLength:  sF.PlayTime,
		SongScore:   sF.Score,
	}

	handleKey := func(k tcell.Key) {
		speaker.Lock()
		defer speaker.Unlock()

		switch k {
		case tcell.KeyEnter:
			fallthrough
		case tcell.KeyCtrlSpace:
			if !ctrl.Paused {
				timePaused.Store(time.Now())
			} else {
				tP := timePaused.Load().(time.Time)
				pS := playStart.Load().(time.Time)
				playStart.Store(pS.Add(time.Since(tP)))
				timePaused.Store(time.Time{})
			}
			ctrl.Paused = !ctrl.Paused

		case tcell.KeyTAB:
			ps := playStart.Load().(time.Time)

			if ctrl.Paused {
				ps = ps.Add(time.Since(timePaused.Load().(time.Time)))
			}

			skipped.Store(true)
			if err := streamer.Seek(streamer.Len() - 1); err != nil {
				panic(err)
			}
		}
	}

	var hK *tcell.EventKey
	for {
		select {
		case <-done:
			mu.Lock()
			if skipped.Load() == nil {
				sF.updatePlayed(playStart.Load().(time.Time))
			} else {
				sF.updateSkipped(playStart.Load().(time.Time))
			}
			mu.Unlock()
			return
		case <-time.After(100 * time.Millisecond):
			speaker.Lock()
			SongTime <- format.SampleRate.D(streamer.Position()).Round(time.Second)
			speaker.Unlock()
		case hK = <-HotkeyEvent:
			handleKey(hK.Key())
		}
	}
}

func (pI *PlayInfo) computeSkipScore() bool {
	//Compute the lastSkipped scores
	if pI.LastSkipped > lib.LastCompute {
		pI.Score -= 15 * (1 + pI.ConsecutiveSkips)

		if pI.TotalSkips > uint64(math.Floor(lib.AvgSkips)) {
			pI.Score -= 15
		}

		pI.ConsecutiveSkips++

		return false
	}

	if pI.LastSkipped > pI.LastPlayed {
		pI.ConsecutiveSkips = 0
	}

	//give the skipped songs a bit of attrition
	pI.Score += 5
	return true
}

func (pI *PlayInfo) computePlayScore() {
	if pI.LastPlayed < lib.LastCompute {
		pI.ComputesSincePlay++
		pI.Score += 15 * float64(pI.ComputesSincePlay)
	}

	//We've just played the song, so we're going to drop its score somewhat.
	if pI.LastPlayed > lib.LastCompute && pI.Score > lib.AvgScore {
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
