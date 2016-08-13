package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"math"
	"path/filepath"
)

// Creates sync plan
type Planner struct {
	lib            *Library
	playlist       *Playlist
	sinkDir        *SinkDir
	SkippedTracks  int
	SyncingTracks  int
	DeletingTracks int
}

type SinkResult struct {
	Track     Track
	Filename  string
	Performed []IOAction
}

func NewPlanner(lib *Library, pl *Playlist, sinkDir *SinkDir) *Planner {
	return &Planner{
		lib:      lib,
		playlist: pl,
		sinkDir:  sinkDir,
	}
}

func (p *Planner) Start(engine *IOEngine) {
	logrus.Infof("---------- Sync: %s --------------", p.playlist.Name)
	itemLen := len(p.playlist.PlaylistItems)
	results := make([]SinkResult, 0)
	if itemLen == 0 {
		return
	}
	prefixLen := int(math.Ceil(math.Log10(float64(itemLen))))
	skippedTracks := 0
	copyAndRenameActions := make([]IOAction, 0)
	for index, track := range p.playlist.Tracks(p.lib) {
		if len(track.Location) == 0 {
			logrus.Warnf("No File(iCloud): %s", track.Name)
			continue
		}
		extension := filepath.Ext(track.Location)
		// 0001 Track Name.m4a
		// 0002 Track Name2.mp3
		// ...
		newFileName := fmt.Sprintf(fmt.Sprintf("%%0%dd %%s%%s", prefixLen), index+1, escapeFilename(track.Name), extension)
		acts := p.sinkDir.SinkTrack(&track, newFileName)
		if len(acts) == 0 {
			skippedTracks += 1
		}
		copyAndRenameActions = append(copyAndRenameActions, acts...)
		results = append(results, SinkResult{
			Track:     track,
			Filename:  newFileName,
			Performed: acts,
		})
	}
	// Action order
	// Trash -> Copy and Rename -> Update meta.json
	trashUncheckedActions := p.sinkDir.TrashUncheckedTracks(p.lib)
	p.SkippedTracks = skippedTracks
	p.SyncingTracks = len(p.playlist.PlaylistItems) - skippedTracks
	p.DeletingTracks = len(trashUncheckedActions)
	if p.DeletingTracks == 0 && p.SyncingTracks == 0 {
		// nothing changed, skip
		logrus.Debugf("Nothing changed: skipping %s", p.playlist.Name)
		return
	}
	updateMetaActions, err := p.sinkDir.UpdateMeta(results)
	if err != nil {
		logrus.Fatalf("Failed to update metadata for %s", p.playlist.Name)
	}
	for _, act := range trashUncheckedActions {
		engine.Push(act)
	}
	for _, act := range copyAndRenameActions {
		engine.Push(act)
	}
	for _, act := range updateMetaActions {
		engine.Push(act)
	}
	if skippedTracks > 0 {
		logrus.Infof("SKIP Tracks: %d", skippedTracks)
	}
}
