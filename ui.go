package main

import (
	"fmt"
	"github.com/dwood15/mediaplayer/songplayer"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"time"
)

func gridView() *tview.Grid {
	newPrimitive := func(text string) tview.Primitive {
		return tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetText(text)
	}

	view := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDrawFunc(drawTime)

	return tview.NewGrid().
		SetSize(1, 1, 10, 10).
		SetMinSize(10, 10).
		SetBorders(true).
		AddItem(newPrimitive("Header"), 0, 0, 1, 1, 0, 0, false).
		AddItem(view, 1, 1, 1, 1, 0, 0, false)
}

var app = tview.NewApplication()

func launchUI(onInput chan int64, songState chan songplayer.PlayingSong) {
	go refresh(songState)

	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyTAB:
			onInput <- songplayer.SignalSkip
		case tcell.KeyEnter:
			onInput <- songplayer.SignalPause
		case tcell.KeyEsc:
			onInput <- songplayer.SignalExit
			app.Stop()
		}

		return e
	})

	if err := app.SetRoot(gridView(), true).Run(); err != nil {
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

func drawTime(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	timeStr := fmtDuration(songplayer.SongTime.Load().(time.Duration))
	tview.Print(screen, timeStr, x, height/2, width, tview.AlignCenter, tcell.ColorTomato)
	return x, y, width, height
}

func refresh (ss chan songplayer.PlayingSong) {
	tckr := time.NewTicker(25 * time.Millisecond)

	for {
		select {
		case <-tckr.C:
			app.Draw()
		case <-ss:
		}
	}
}