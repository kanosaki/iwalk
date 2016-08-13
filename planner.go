package main

// Creates sync plan
type Planner struct {
	lib      *Library
	playlist *Playlist
	sinkDir  *SinkDir
}

func NewPlanner(lib *Library, pl *Playlist, sinkDir *SinkDir) *Planner {
	return &Planner{
		lib:      lib,
		playlist: pl,
		sinkDir:  sinkDir,
	}
}

func (p *Planner) Start(engine *IOEngine) {
	for index, track := range p.playlist.Tracks(p.lib) {
		p.sinkDir.SinkTrack(track, index)
	}
}
