package main

import (
	"fmt"
	"github.com/dwood15/mediaplayer/songs"
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
		SetSize(1, 1, 15, 20).
		SetMinSize(10, 10).
		SetBorders(true).
		AddItem(newPrimitive("Header"), 0, 0, 1, 1, 5, 5, false).
		AddItem(view, 1, 1, 1, 1, 5, 5, false)
}


var app  = tview.NewApplication()

func launchUI() {

	go refresh()

	app.SetInputCapture(musicPlayerSignal)
	grid := gridView()
	if err := app.SetRoot(grid, true).Run(); err != nil {
		panic(err)
	}
}

func musicPlayerSignal(e *tcell.EventKey) *tcell.EventKey {
	switch e.Key() {
	case tcell.KeyTAB:
		songs.PlayerSignal <- songs.SignalSkip
	case tcell.KeyEnter:
		songs.PlayerSignal <- songs.SignalPause
	case tcell.KeyEsc:
		songs.PlayerSignal <- songs.SignalExit
		app.Stop()
	}

	return e
}

func fmtDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d", m, s)
}

func drawTime(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	timeStr := fmtDuration(songs.SongTime.Load().(time.Duration))
	tview.Print(screen, timeStr, x, height/2, width, tview.AlignCenter, tcell.ColorTomato)
	return x, y, width, height
}

func refresh() {
	tckr := time.NewTicker(25 * time.Millisecond)

	for {
		select {
		case <-tckr.C:
			app.Draw()
		case <-songs.SongState:
		}
	}
}
