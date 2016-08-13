package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"math"
	"path/filepath"
)

// Creates sync plan
type Planner struct {
	lib          *Library
	playlist     *Playlist
	sinkDir      *SinkDir
	planResult   []SinkResult
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
		p.planResult = results
		return
	}
	prefixLen := int(math.Ceil(math.Log10(float64(itemLen))))
	skippedTracks := 0
	for index, track := range p.playlist.Tracks(p.lib) {
		extension := filepath.Ext(track.Location)
		// 0001 Track Name.m4a
		// 0002 Track Name2.mp3
		// ...
		newFileName := fmt.Sprintf(fmt.Sprintf("%%0%dd %%s%%s", prefixLen), index + 1, escapeFilename(track.Name), extension)
		acts := p.sinkDir.SinkTrack(&track, newFileName)
		if len(acts) == 0 {
			skippedTracks += 1
		}
		for _, act := range acts {
			engine.Push(act)
		}
		results = append(results, SinkResult{
			Track:     track,
			Filename:  newFileName,
			Performed: acts,
		})
	}
	for _, act := range p.sinkDir.TrashUncheckedTracks(p.lib) {
		engine.Push(act)
	}
	if skippedTracks > 0 {
		logrus.Infof("SKIP Tracks: %d", skippedTracks)
	}
	p.planResult = results
}

func (p *Planner) UpdateMetadata() error {
	if *argDryRun {
		logrus.Infof("DRYRUN: Skipping UpdateMetadata")
		return nil
	} else {
		err := p.sinkDir.UpdateMeta(p.planResult)
		if err != nil {
			logrus.Errorf("Faild to update metadata at %s: %s", p.sinkDir.Path, err)
		}
		return err
	}
}
