package watcher

import (
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/uddinArsalan/ferry/types"
)

type Watcher struct {
	eventChan chan types.Event
	watcher   *fsnotify.Watcher
}

func NewWatcher(eventChan chan types.Event) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Error in creating watcher")
		return nil, err
	}
	return &Watcher{
		eventChan: eventChan,
		watcher:   watcher,
	}, nil
}

func (w *Watcher) WatchFile(dirPath string, groupID string) {
	dirs := make(chan string)
	go func() {
		defer close(dirs)
		if err := w.recursiveDir(dirPath, dirs); err != nil {
			log.Printf("Error in recursive dir: %v", err)
			return
		}
	}()

	log.Printf("DIR : %v", dirs)

	for dir := range dirs {
		if err := w.watcher.Add(dir); err != nil {
			log.Printf("Error in adding directory %v: %v", dir, err)
			return
		}
	}

	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Chmod) {
					continue
				}
				if event.Has(fsnotify.Create) {
					info, err := os.Stat(event.Name)
					if err != nil {
						continue
					}
					if info.IsDir() {
						if err := w.watcher.Add(event.Name); err != nil {
							log.Printf("Error in adding directory %v: %v", event.Name, err)
							return
						}
					}
				}
				w.eventChan <- types.Event{
					Path:    event.Name,
					Op:      types.EventType(event.Op),
					GroupID: groupID,
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
