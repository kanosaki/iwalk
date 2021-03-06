package main

import (
	"github.com/DHowett/go-plist"
	"os"
	"strconv"
	"time"
)

// from https://github.com/ericdaugherty/itunesexport-go/blob/master/library.go

type Library struct {
	MajorVersion        int `plist:"Major Version"`
	MinorVersion        int `plist:"Minor Version"`
	Date                time.Time
	ApplicationVersion  int
	Features            int
	ShowContentRating   bool   `plist:"Show Content Ratings"`
	MusicFolder         string `plist:"Music Folder"`
	LibraryPersistentId string `plist:"Library Persistent ID"`
	Tracks              map[string]Track
	Playlists           []Playlist
	PlaylistMap         map[string]Playlist
	persistentMap       map[string]*Track
}

type Track struct {
	TrackId             int `plist:"Track ID"`
	Name                string
	Artist              string
	AlbumArtist         string `plist:"Album Artist"`
	Composer            string
	Album               string
	Genre               string
	Kind                string
	Size                int
	TotalTime           int `plist:"Total Time"`
	TrackNumber         int `plist:"Track Number"`
	Year                int
	DateModified        time.Time `plist:"Date Modified"`
	DateAdded           time.Time `plist:"Date Added"`
	BitRate             int       `plist:"Bit Rate"`
	SampleRate          int       `plist:"Sample Rate"`
	PlayCount           int       `plist:"Play Count"`
	PlayDate            int       `plist:"Play Date"`
	PlayDateUTC         time.Time `plist:"Play Date UTC"`
	SkipCount           int       `plist:"Skip Count"`
	SkipDate            time.Time `plist:"Skip Date"`
	Rating              int
	AlbumRating         int    `plist:"Album Rating"`
	AlbumRatingComputed bool   `plist:"Album Rating Computed"`
	ArtworkCount        int    `plist:"Artwork Count"`
	PersistentId        string `plist:"Persistent ID"`
	TrackType           string `plist:"Track Type"`
	Location            string
	FileFolderCount     int `plist:"File Folder Count"`
	LibraryFolderCount  int `plist:"Library Folder Count"`
}

type Playlist struct {
	Name                 string
	Master               bool
	PlaylistId           int    `plist:"Playlist ID"`
	PlaylistPersistentId string `plist:"Playlist Persistent ID"`
	DistinguishedKind    int    `plist:"Distinguished Kind"`
	Visible              bool
	AllItems             bool           `plist:"All Items"`
	SmartInfo            []byte         `plist:"Smart Info"`
	SmartCriteria        []byte         `plist:"Smart Criteria"`
	PlaylistItems        []PlaylistItem `plist:"Playlist Items"`
}

type PlaylistItem struct {
	TrackId int `plist:"Track ID"`
}

func LoadLibrary(fileLocation string) (returnLibrary *Library, err error) {

	if _, statErr := os.Stat(fileLocation); os.IsNotExist(statErr) {
		err = statErr
		return
	}

	file, pathErr := os.Open(fileLocation)
	if pathErr != nil {
		err = pathErr
		return
	}
	defer file.Close()

	decoder := plist.NewDecoder(file)

	var library Library
	decodeErr := decoder.Decode(&library)
	if decodeErr != nil {
		err = decodeErr
		return
	}

	library.PlaylistMap = make(map[string]Playlist, len(library.Playlists))
	for _, value := range library.Playlists {
		library.PlaylistMap[value.Name] = value
	}

	return &library, err
}

func (playlist *Playlist) Tracks(library *Library) []Track {
	tracks := make([]Track, 0, len(playlist.PlaylistItems))
	for _, item := range playlist.PlaylistItems {
		track, ok := library.Tracks[strconv.FormatInt(int64(item.TrackId), 10)]
		if ok {
			tracks = append(tracks, track)
		}
	}
	return tracks
}
