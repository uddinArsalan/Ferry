package types

import "github.com/fsnotify/fsnotify"

type EventType = fsnotify.Op

const (
	EventCreate = fsnotify.Create
	EventRename = fsnotify.Rename
	EventRemove = fsnotify.Remove
)

type Event struct {
	Path    string
	Op      EventType
	GroupID string
}
