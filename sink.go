package main

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"os"
	"path"
	"time"
	"strconv"
)

const META_JSON_FILENAME = "meta.json"
const META_JSON_TEMP_FILENAME = "meta.temp.json"

type Sink struct {
	Path string
}

type SinkDir struct {
	Path               string                `json:"-"`
	CheckedTracks      map[string]bool       `json:"-"`
	Tracks             map[string]*TrackMeta `json:"tracks"`
	OriginPlaylistID   string                `json:"origin_playlist_id"`
	OriginPlaylistName string                `json:"origin_playlist_name"`
}

type TrackMeta struct {
	OriginID           string    `json:"origin_id"`
	OriginPersistentID string    `json:"origin_persistent_id"`
	FileName           string    `json:"filename"`
	ModifiedTime       time.Time `json:"modified_time"`
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
	if err != nil {
		return nil, err
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	var ret SinkDir
	err = decoder.Decode(&ret)
	if err != nil {
		return nil, err
	} else {
		ret.Path = dirPath
		ret.CheckedTracks = make(map[string]bool)
		return &ret, nil
	}
}

func (s *Sink) createSinkDir(dirPath string) (*SinkDir, error) {
	if *argDryRun {
		logrus.Infof("DRYRUN: Creating new sink dir: %s", dirPath)
	} else {
		logrus.Infof("Creating new sink dir: %s", dirPath)
		os.MkdirAll(dirPath, 0775)
	}
	return &SinkDir{
		CheckedTracks: make(map[string]bool),
		Path:          dirPath,
		Tracks:        make(map[string]*TrackMeta),
	}, nil
}

func (s *SinkDir) copyFromLocal(track *Track, sinkPath string) IOAction {
	localPath := normalizeLocation(track.Location)
	return NewCopy(localPath, sinkPath, track)
}

func (s *SinkDir) SinkTrack(track *Track, fileName string) []IOAction {
	trackId := track.PersistentId
	meta, previouslyExists := s.Tracks[trackId]
	sinkPath := path.Join(s.Path, fileName)
	if previouslyExists {
		defer func() {
			s.CheckedTracks[trackId] = true
		}()
		prevPath := path.Join(s.Path, meta.FileName)
		if meta.ModifiedTime.Before(track.DateModified) {
			// has update
			if isWritable(prevPath) {
				logrus.Infof("-- UPDATE: %s (%s -> %s)", track.Name, meta.FileName, fileName)
				return []IOAction{
					NewDelete(prevPath),
					s.copyFromLocal(track, sinkPath),
				}
			} else {
				logrus.Warnf("-- UPDATE: %s (could not delete old file %s)", track.Name, meta.FileName)
				return []IOAction{
					s.copyFromLocal(track, sinkPath),
				}
			}
		} else {
			if prevPath == sinkPath {
				logrus.Debugf("-- NOP   : %s", track.Name)
				return []IOAction{} // nop
			}
			// perform move
			if isWritable(prevPath) {
				logrus.Infof("-- RENAME: %s (%s -> %s)", track.Name, meta.FileName, fileName)
				return []IOAction{NewRename(prevPath, sinkPath)}
			} else {
				logrus.Infof("-- COPY**: %s (Unable to find previous file: %s)", track.Name, meta.FileName)
				return []IOAction{s.copyFromLocal(track, sinkPath)}
			}
		}
	} else {
		// perform copy
		logrus.Infof("-- COPY  : %s", track.Name)
		return []IOAction{s.copyFromLocal(track, sinkPath)}
	}
}

func (s *SinkDir) TrashUncheckedTracks(lib *Library) []IOAction {
	ret := make([]IOAction, 0)
	for trackId, trackMeta := range s.Tracks {
		if _, ok := s.CheckedTracks[trackId]; !ok {
			if originTrack, ok := lib.Tracks[trackMeta.OriginID]; ok {
				logrus.Infof("-- DELETE: %s (%s)", originTrack.Name, trackMeta.FileName)
			} else {
				logrus.Infof("-- DELETE: %s", trackMeta.FileName)
			}
			trashPath := path.Join(s.Path, trackMeta.FileName)
			if isWritable(trashPath) {
				ret = append(ret, NewDelete(trashPath))
			} else {
				logrus.Warnf("Trash failed: %s(ID: %s, pID: %s)", trackMeta.FileName, trackMeta.OriginID, trackMeta.OriginPersistentID)
			}
		}
	}
	return ret
}

func (s *SinkDir) UpdateMeta(sinkResults []SinkResult) ([]IOAction, error) {
	s.Tracks = make(map[string]*TrackMeta)
	for _, result := range sinkResults {
		trackId := result.Track.PersistentId
		s.Tracks[trackId] = &TrackMeta{
			OriginID:           strconv.Itoa(result.Track.TrackId),
			OriginPersistentID: result.Track.PersistentId,
			FileName:           result.Filename,
			ModifiedTime:       result.Track.DateModified,
		}
	}
	metaPath := path.Join(s.Path, META_JSON_FILENAME)
	data, err := json.Marshal(s)
	if err != nil {
		return []IOAction{}, err
	}
	return ([]IOAction{
		NewWriteFileAction(metaPath, path.Join(s.Path, META_JSON_TEMP_FILENAME), data),
	}), nil
}
