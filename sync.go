package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
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
		lib:           itunesLib,
		sink:          sink,
		syncPlaylists: playlists,
	}
	return ctx.Start()
}

func (c *SyncContext) Start() (err error) {
	engine := NewIOEngine()
	planners := make([]*Planner, 0, len(c.syncPlaylists))
	for _, playlistName := range c.syncPlaylists {
		sinkDir, err := c.sink.OpenSinkDir(playlistName, true)
		if err != nil {
			return err
		}
		playlist, ok := c.lib.PlaylistMap[playlistName]
		if !ok {
			return fmt.Errorf("Playlist '%s' not found in iTuens library", playlistName)
		}
		planner := NewPlanner(c.lib, &playlist, sinkDir)
		planner.Start(engine)
		planners = append(planners, planner)
	}
	logrus.Infof("Commiting..")
	err = engine.Commit()
	if err != nil {
		return
	}
	for _, planner := range planners {
		err = planner.UpdateMetadata()
		if err != nil {
			logrus.Errorf("Failed to UpdateMetadata: %s", err)
		}
	}
	return
}
