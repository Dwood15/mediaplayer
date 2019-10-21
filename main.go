package main

import (
	"fmt"
	"github.com/dwood15/mediaplayer/songs"
)

func main() {
	l := songs.GetLibrary()

	l.Play()
	l.Play()
	songs.PersistLibCache()

	fmt.Println("playing complete")
}
