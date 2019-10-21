package main

import (
	"fmt"
	"github.com/dwood15/mediaplayer/songs"
)

func main() {
	_ = songs.GetLibrary()

	songs.PersistLibCache()


	fmt.Println("playing complete")
}
