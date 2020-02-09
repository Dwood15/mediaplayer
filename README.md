# Golang Media Player

A command line go-based media player with a simple media library.

## Motivation

I have become frustrated with the absolutely garbage state of typical media player shuffle solutions.

The ONLY goal of this player is to shuffle songs and provide a sane algorithm for when new ones are played.

I have a media library >80GB, and no media player provides adequate shuffle for continuous play across multiple sessions.

## Status

- Play functionality for mp3's is built. (automatically plays the first song)
- Basic rating system is built. 
- Loads mp3's within the `music_dir`, specified in config.json
- Plays a 'playlist' of songs, saves lib to cache, then exits.
- Basic skipping is 'built' (not implemented)

## Goals

- Don't repeat songs too much
- If songs are skipped, lower their priority
- Songs that get played less are more highly prioritized
- Don't play songs less than a minute and a half in length

## Usage

- EXPECT BUGS 
- Ensure you have a music folder in your user directory, and if not, create a config.json in the project directory:
```json
{
  "music_dir": "Full/Path/To/Your/Music/Dir",
  "max_playlist_size": 25
}
```
- When the application is built and ran, it will consume as much of your system resources as it can, in order to chew through your music folder ASAP.
 Running on my (very fast, very powerful) machine took 3m 28.5s
 
- Due to the limitations of the libraries beep depends on,  only select kinds of MP3 files are supported.

- To build and run (linux): `go build && ./mediaplayer`

## TODO
- Implement keyboard input (lol) 
- Implement a console ui, such as: https://github.com/gcla/gowid
- Optimize first boot so it won't bring systems to their knees
- Custom file / database structure (https://cstack.github.io/db_tutorial), maybe fs or streaming for true network support
