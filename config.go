package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type config struct {
	MusicDir        string `json:"music_dir"` //MusicDir is the directory where the
	MaxPlaylistSize int    `json:"max_playlist_size"`
}

func loadConfig() config {
	var cfg config

	f, err := os.Open("config.json")

	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}

		h, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}

		cfg.MusicDir = h + "/Music"
		cfg.MaxPlaylistSize = 25

		f, err := os.Create("config.json")
		if err != nil {
			panic(err)
		}

		newConfig, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			panic(err)
		}

		if _, err = f.Write(newConfig); err != nil {
			panic(err)
		}

		f.Close()
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	f.Close()

	if err = json.Unmarshal(b, &cfg); err != nil {
		panic(err)
	}

	return cfg
}
