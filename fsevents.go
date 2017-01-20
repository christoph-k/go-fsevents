package fsevents

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FSEventType ...
type FSEventType int

// FSEventCreate ...
const FSEventCreate FSEventType = 0

// FSEventDelete ...
const FSEventDelete FSEventType = 1

// FSEventModify ...
const FSEventModify FSEventType = 2

// FSEvent ...
type FSEvent struct {
	Path             string
	EventType        FSEventType
	FileInfo         os.FileInfo
	PreviousFileInfo os.FileInfo
}

// Watcher ...
type Watcher struct {
	Path      string
	Interval  time.Duration
	EventChan chan FSEvent
	index     map[string]os.FileInfo
	active    bool
	mux       sync.Mutex
}

// NewWatcher ...
func NewWatcher(path string, interval time.Duration) (*Watcher, error) {
	var err error
	w := &Watcher{}
	if interval != 0 {
		w.Interval = interval
	} else {
		return nil, fmt.Errorf("interval must be > 0")
	}
	w.Path, err = filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get abspath for %v", path)
	}
	w.EventChan = make(chan FSEvent)
	w.index = make(map[string]os.FileInfo)
	err = filepath.Walk(w.Path, func(path string, info os.FileInfo, err error) error {
		w.index[path] = info
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("initial scan failed (%v)", err)
	}
	go w.watch()
	w.mux.Lock()
	defer w.mux.Unlock()
	w.active = true
	return w, nil
}

// Stop ...
func (w *Watcher) Stop() {
	close(w.EventChan)
	w.mux.Lock()
	defer w.mux.Unlock()
	w.active = false
}

func (w *Watcher) watch() {
	w.mux.Lock()
	defer w.mux.Unlock()
	for w.active {
		fsstate := make(map[string]os.FileInfo)
		err := filepath.Walk(w.Path, func(path string, info os.FileInfo, err error) error {
			if info == nil {
				return nil
			}
			fsstate[path] = info
			return nil
		})
		if err != nil {
			panic(err)
		}

		w.checkDelete(fsstate)
		w.checkCreate(fsstate)
		w.checkModify(fsstate)

		time.Sleep(w.Interval)
	}
}

func (w *Watcher) checkDelete(fsstate map[string]os.FileInfo) {
	for oldkey := range w.index {
		if _, ok := fsstate[oldkey]; !ok {
			e := FSEvent{
				Path:             oldkey,
				EventType:        FSEventDelete,
				FileInfo:         nil,
				PreviousFileInfo: w.index[oldkey],
			}
			w.EventChan <- e
			delete(w.index, oldkey)
		}
	}
}

func (w *Watcher) checkCreate(fsstate map[string]os.FileInfo) {
	for newkey, newval := range fsstate {
		if _, ok := w.index[newkey]; !ok {
			e := FSEvent{
				Path:             newkey,
				EventType:        FSEventCreate,
				FileInfo:         newval,
				PreviousFileInfo: nil,
			}
			w.EventChan <- e
			w.index[newkey] = newval
		}
	}
}

func (w *Watcher) checkModify(fsstate map[string]os.FileInfo) {
	for newkey, newval := range fsstate {
		oldval := w.index[newkey]
		if oldval.IsDir() != newval.IsDir() || oldval.ModTime() != newval.ModTime() ||
			oldval.Mode() != newval.Mode() || oldval.Size() != newval.Size() {
			e := FSEvent{
				Path:             newkey,
				EventType:        FSEventModify,
				FileInfo:         newval,
				PreviousFileInfo: oldval,
			}
			w.EventChan <- e
			w.index[newkey] = newval
		}
	}
}
