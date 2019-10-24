package main

import (
	"fmt"
	"github.com/dwood15/mediaplayer/songs"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"time"
)

var (
	app  = tview.NewApplication()
	view = tview.NewBox().SetDrawFunc(drawTime)
)

func launchUI() {
	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEsc {
			songs.PlayerSignal<-songs.SignalExit
			app.Stop()
		}
		return e
	})

	view.SetBackgroundColor(tcell.ColorBlack)
	view.SetInputCapture(musicPlayerSignal)

	go refresh()

	if err := app.SetRoot(view, false).Run(); err != nil {
		panic(err)
	}
}

func musicPlayerSignal(e *tcell.EventKey) *tcell.EventKey {
	var toSignal int
	switch e.Key() {
	case tcell.KeyTAB:
		toSignal = songs.SignalSkip
	case tcell.KeyEnter:
		toSignal = songs.SignalPause
	}

	if toSignal > 0 {
		songs.PlayerSignal <- toSignal
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
	tview.Print(screen, timeStr, x, height/2, width, tview.AlignCenter, tcell.ColorDarkBlue)
	return 0, 0, 0, 0
}

func refresh() {
	for {
		select {
		case <-time.After(50 * time.Millisecond):
			app.Draw()
		case <-songs.SongState:

		}
	}
}

