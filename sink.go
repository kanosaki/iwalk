package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

const META_JSON_FILENAME = "meta.json"

type Sink struct {
	Path string
}

type SinkDir struct {
	Path               string                `json:"-"`
	Tracks             map[string]*TrackMeta `json:"tracks"`
	OriginPlaylistID   string                `json:"origin_playlist_id"`
	OriginPlaylistName string                `json:"origin_playlist_name"`
}

type TrackMeta struct {
	OriginID string `json:"origin_id"`
	FileName string `json:"filename"`
}

func NewSink(sinkPath string) (*Sink, error) {
	// TODO: validate sinkPath
	return &Sink{
		Path: sinkPath,
	}, nil
}

// TODO: Should cache result?
func (s *Sink) OpenSinkDir(name string, createIfAbsent bool) (*SinkDir, error) {
	dirPath := path.Join(s.Path, name)
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		if createIfAbsent {
			return s.createSinkDir(dirPath)
		} else {
			return nil, fmt.Errorf("Sink directory %s does not exist!", dirPath)
		}
	} else {
		return s.openSinkDirContents(dirPath)
	}
}

// dirPath should be valid directory path
func (s *Sink) openSinkDirContents(dirPath string) (*SinkDir, error) {
	metaPath := path.Join(dirPath, META_JSON_FILENAME)
	f, err := os.Open(metaPath)
	if os.IsNotExist(err) {
		return nil, err
	} else {
		decoder := json.NewDecoder(f)
		var ret *SinkDir
		err = decoder.Decode(ret)
		if err != nil {
			ret.Path = dirPath
			return ret, nil
		} else {
			return nil, err
		}
	}
}

func (s *Sink) createSinkDir(dirPath string) (*SinkDir, error) {
	return &SinkDir{
		Path:   dirPath,
		Tracks: make(map[string]*TrackMeta),
	}, nil
}

func (s *SinkDir) SinkTrack(track *Track, index int) IOAction {
	meta, previouslyExists := s.Tracks[string(track.TrackId)]
	if previouslyExists {
		// perform move

	} else {
		// perform copy

	}
}
