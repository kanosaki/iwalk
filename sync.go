package main

import (
	"fmt"
)

type SyncContext struct {
	lib           *Library
	sink          *Sink
	syncPlaylists []string
}

func startSync(libPath, targetDir string, playlists []string) error {
	itunesLib, err := LoadLibrary(libPath)
	if err != nil {
		return err
	}
	sink, err := NewSink(targetDir)
	if err != nil {
		return err
	}
	ctx := &SyncContext{
		lib: itunesLib,
		sink: sink,
		syncPlaylists: playlists,
	}
	return ctx.Start()
}

func (c *SyncContext) Start() error {
	engine := NewIOEngine()
	for _, playlistName := range c.syncPlaylists {
		sinkDir, err := c.sink.OpenSinkDir(playlistName, true)
		if err != nil {
			return err
		}
		playlist, ok := c.lib.PlaylistMap[playlistName]
		if !ok {
			return fmt.Errorf("Playlist '%s' not found in iTuens library", playlistName)
		}
		planner := NewPlanner(playlist, sinkDir)
		planner.Start(engine)
	}
	return engine.Commit()
}
