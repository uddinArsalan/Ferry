package watcher

import (
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/uddinArsalan/ferry/types"
)

type Watcher struct {
	watcher *fsnotify.Watcher
}

func NewWatcher() (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Error in creating watcher")
		return nil, err
	}
	return &Watcher{
		watcher: watcher,
	}, nil
}

func (w *Watcher) WatchFile(dirPath string, eventChan chan types.Event) {
	dirs := make(chan string)
	go func() {
		defer close(dirs)
		if err := w.recursiveDir(dirPath, dirs); err != nil {
			log.Printf("Error in recursive dir: %v", err)
			return
		}
	}()

	for dir := range dirs {
		if err := w.watcher.Add(dir); err != nil {
			log.Printf("Error in adding directory %v: %v", dir, err)
			return
		}
	}

	go func() {
		defer close(eventChan)
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Chmod) {
					continue
				}
				eventChan <- types.Event{
					Path: event.Name,
					Op:   types.EventType(event.Op),
				}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

}

func (w *Watcher) recursiveDir(dirPath string, dirs chan string) error {
	dirs <- dirPath
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err = w.recursiveDir(dirPath+"/"+entry.Name(), dirs); err != nil {
				return err
			}
		}
	}
	return nil
}

// multiple peers will be connected each peer has some directorys they want to sync with
// other peers that also has same directory , root dir will differ, sync group will be used
// sync group contains group id they listen to each group contains other peers and for one peer
// the root
