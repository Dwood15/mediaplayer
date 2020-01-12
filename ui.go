package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	"github.com/dwood15/mediaplayer/songplayer"
)

func (u *UIController) gridView() *tview.Grid {
	newPrimitive := func(text string) tview.Primitive {
		return tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetText(text)
	}

	view := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDrawFunc(u.drawTime)

	return tview.NewGrid().
		SetSize(1, 1, 10, 10).
		SetMinSize(10, 10).
		SetBorders(true).
		AddItem(newPrimitive("Header"), 0, 0, 1, 1, 0, 0, false).
		AddItem(view, 1, 1, 1, 1, 0, 0, false)
}

var app = tview.NewApplication()

type UIController struct {
	SongState    *atomic.Value
	InputChan    chan int64
	currentState songplayer.PlayingSong
}

func (u *UIController) launchUI() {
	go func() {
		tckr := time.NewTicker(25 * time.Millisecond)

		for {
			select {
			case <-tckr.C:
				u.currentState = u.SongState.Load().(songplayer.PlayingSong)
				app.Draw()
			}
		}
	}()

	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyTAB:
			u.InputChan <- songplayer.SignalSkip
		case tcell.KeyEnter:
			u.InputChan <- songplayer.SignalPause
		case tcell.KeyEsc:
			u.InputChan <- songplayer.SignalExit
			app.Stop()
		}

		return e
	})

	if err := app.SetRoot(u.gridView(), true).Run(); err != nil {
		panic(err)
	}
}

func fmtDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d", m, s)
}

func (u *UIController) drawTime(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	timeStr := fmtDuration(u.currentState.SongTime) + " / " + fmtDuration(u.currentState.SongLength)
	ht := height / 2
	tview.Print(screen, timeStr, x, ht, width, tview.AlignCenter, tcell.ColorTomato)
	tview.Print(screen, u.currentState.CurrentSong, x, ht+1, width, tview.AlignCenter, tcell.ColorTomato)
	return x, y, width, height
}
