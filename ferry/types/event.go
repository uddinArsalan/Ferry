package types

import "github.com/fsnotify/fsnotify"

type EventType fsnotify.Op

const (
	CREATE = fsnotify.Create
	RENAME = fsnotify.Rename
	REMOVE = fsnotify.Remove
) 

type Event struct {
	Path string
	Op   EventType
}
