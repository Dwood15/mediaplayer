# Golang Media Player

A command line go-based media player with a simple media library.

## Motivation

Frustrated with the absolutely garbage state of typical media player shuffle solutions?

The ONLY goal of this player is to shuffle songs and provide a sane algorithm for when new ones are played.

I have a media library >80GB, and no media player provides adequate shuffle for continuous play across multiple sessions.

## Status

- Play functionality for mp3's is built.
- Basic rating system is built.
- Basic skipping is built
- Loads mp3's within ./data/ folder,
- Plays a 'playlist' of songs, saves lib to cache, then exits.

## Goals

- Don't repeat songs too much
- If songs are skipped, lower their priority
- Songs that get played less are more highly prioritized
- Don't play songs less than a minute and a half in length

## Usage

- EXPECT BUGS 