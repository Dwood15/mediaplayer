package songplayer

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

type (
	PlayInfo struct {
		Score             uint64 `json:"score,omitempty"`
		TotalSkips        uint64 `json:"total_skips,omitempty"`
		ConsecutiveSkips  uint   `json:"consecutive_skips,omitempty"`
		LastSkipped       int64  `json:"last_skipped_time,omitempty"`
		LastPlayed        int64  `json:"last_played_time,omitempty"`
		TotalPlays        uint64 `json:"total_plays,omitempty"`
		ComputesSincePlay uint8  `json:"computes_since_last_play,omitempty"`
	}
	SongFile struct {
		FileName string        `json:"file_name,omitempty"`
		PlayTime time.Duration `json:"play_time,omitempty"`
		PlayInfo
	}
	PlayingSong struct {
		SongScore   uint64
		SongLength  time.Duration
		CurrentSong string
	}
)

const (
	SignalPause int64 = iota + 1
	SignalPlay
	SignalSkip
	SignalExit
	SignalSongComplete
)

//Cross-goroutine helpers

//SongState indicates what song is playing
var (
	SongState = make(chan PlayingSong)

	//SongTime indicates how far we've progressed in the song
	SongTime atomic.Value

	//PlayerSignal signals input state from the ui to the player
	PlayerSignal = make(chan int64)

	//playMu is for ensuring only one song is playing
	playMu sync.Mutex
	//Concurrency-safe containers for playback crosstalk.
	playStart  atomic.Value
	timePaused atomic.Value
	format     beep.Format
)

func init() {
	SongTime.Store(time.Duration(0))
}

func (sF *SongFile) initFile() (s beep.StreamSeeker) {
	//Load the song file
	f, err := os.Open(sF.FileName)
	if err != nil {
		panic(err)
	}

	//load beep's StreamSeeker
	if s, format, err = mp3.Decode(f); err != nil {
		panic(err)
	}

	//Any higher precision than this ends with the songs playing much faster than intended, for some reason.
	//Not particularly inclined to debug the intricacies of Beep and its child packages.
	snr := format.SampleRate.N(time.Second / 2)

	//Rather than muck about with funky math for different song sample rates,
	//let's just initialize it to the exact format every time - we're only gonna
	//be playing one song at a time, anyway.
	if err = speaker.Init(format.SampleRate, snr); err != nil {
		panic(err)
	}

	//This seems to decrease performance quite a bit, leaving it commented out for now.
	//buf := beep.NewBuffer(format)
	//buf.Append(s)
	//s = buf.Streamer(0, buf.Len())

	fmt.Println("sending song state to server process")
	//Signal to the ui what's playing. Perhaps an atomic.Value would be better?
	SongState <- PlayingSong{
		CurrentSong: path.Base(sF.FileName),
		SongLength:  sF.PlayTime,
		SongScore:   sF.Score,
	}

	return s
}

//Play locks the current goroutine/thread until an interrupt
func (sF *SongFile) play() (shouldExit bool) {
	playMu.Lock()

	fmt.Println("initializing song file")

	s := sF.initFile()

	fmt.Println("song file initialized, initializing player")

	var skipped atomic.Value
	skipped.Store(false)

	//So we can pause songs
	ctrl := &beep.Ctrl{
		Paused: false,
		Streamer: beep.Seq(s, beep.Callback(func() {
			wasSkipped := skipped.Load().(bool)
			if wasSkipped {
				return
			}

			PlayerSignal <- SignalSongComplete
		})),
	}

	//fmt.Println("initiating play for song: " + sF.FileName)

	timePaused.Store(time.Time{})
	//So we know when the song started.
	playStart.Store(time.Now())

	speaker.Play(ctrl)

	var plyrSig int64
	tkr := time.NewTicker(75 * time.Millisecond)

	for {
		select {
		case <-tkr.C:
			SongTime.Store(format.SampleRate.D(s.Position()))
		case plyrSig = <-PlayerSignal:
			//Signal the exit here, which will cause the done func up above to trigger and send
			//the signalComplete signal. Hopefully, out of order event reception doesn't happen super often
			switch plyrSig {
			case SignalSkip:
				speaker.Clear()
				goto closeShop
			case SignalExit:
				shouldExit = true
				skipped.Store(false)
			case SignalPause, SignalPlay:
				sF.togglePause(ctrl)
			case SignalSongComplete:
				goto closeShop
			}
			plyrSig = 0
		}
	}

closeShop:
	tkr.Stop()
	playMu.Unlock()
	skpd := plyrSig == SignalSkip
	skipped.Store(skpd)
	sF.onFinish(ctrl, skpd)
	return
}

func (sF *SongFile) togglePause(ctrl *beep.Ctrl) {
	pAt := time.Now()
	if ctrl.Paused {
		tP := timePaused.Load().(time.Time)
		pS := playStart.Load().(time.Time)
		playStart.Store(pS.Add(time.Since(tP)))
		pAt = time.Time{}
	} else {
		timePaused.Store(pAt)
	}

	speaker.Lock()
	ctrl.Paused = !ctrl.Paused
	speaker.Unlock()
}

func (sF *SongFile) onFinish(ctrl *beep.Ctrl, skipped bool) {
	speaker.Lock()
	ps := playStart.Load().(time.Time)
	if ctrl.Paused {
		ps = ps.Add(time.Since(timePaused.Load().(time.Time)))
	}
	speaker.Unlock()

	lib.mu.Lock()

	sF.update(ps, skipped)

	lib.mu.Unlock()
	SongTime.Store(time.Duration(0))
}

func (sF *SongFile) loadPlayTime() error {
	f, err := os.Open(sF.FileName)
	if err != nil {
		panic(err)
	}

	streamer, _fmt, err := mp3.Decode(f)
	if err != nil {
		return err
	}

	sF.PlayTime = _fmt.SampleRate.D(streamer.Len())
	_ = streamer.Close()
	return nil
}

func (sF *SongFile) update(s time.Time, skipped bool) {
	if skipped {
		sF.ConsecutiveSkips++
		sF.TotalSkips++
		sF.ComputesSincePlay = 0
		sF.LastSkipped = time.Now().Unix()
		lib.NumSkips++
		lib.TimePlayed += time.Since(s)
		return
	}
	sF.TotalPlays++
	sF.LastPlayed = time.Now().Unix()
	sF.ConsecutiveSkips = 0
	sF.ComputesSincePlay = 0
	lib.NumPlays++
	lib.TimePlayed += time.Since(s)

}

//computeSkipScore returns false if we should compute PlayScore
func (pI *PlayInfo) computeSkipScore() bool {
	//Compute the lastSkipped scores
	if pI.LastSkipped > lib.LastCompute {
		pI.Score -= 15 * uint64(1+pI.ConsecutiveSkips)

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
		pI.Score += 15 * uint64(pI.ComputesSincePlay)
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
		pI.Score += uint64(math.Floor(float64(len(lib.Songs)) * rand.Float64()))
	}

	//[0, 5]
	pI.Score += uint64(math.Floor(5 * rand.Float64()))

	if !pI.computeSkipScore() {
		return
	}

	pI.computePlayScore()
}
