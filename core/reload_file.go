package core

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var reloadFileChangeOps = fsnotify.Write | fsnotify.Create | fsnotify.Rename | fsnotify.Remove | fsnotify.Chmod

type ReloadFileWatcher struct {
	Changes <-chan struct{}
	Errors  <-chan error
	close   func()
}

func (w *ReloadFileWatcher) Close() {
	if w != nil && w.close != nil {
		w.close()
	}
}

func WatchReloadFile(path string) (*ReloadFileWatcher, error) {
	path = filepath.Clean(path)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(filepath.Dir(path)); err != nil {
		watcher.Close()
		return nil, err
	}

	changes := make(chan struct{}, 1)
	errors := make(chan error, 1)
	done := make(chan struct{})

	go func() {
		defer close(changes)
		defer close(errors)
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if reloadFileEventMatches(event, path) {
					signalReloadFileChange(changes)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				if err != nil {
					signalReloadFileError(errors, err)
				}
			case <-done:
				return
			}
		}
	}()

	return &ReloadFileWatcher{
		Changes: changes,
		Errors:  errors,
		close: func() {
			close(done)
		},
	}, nil
}

func reloadFileEventMatches(event fsnotify.Event, path string) bool {
	return event.Op&reloadFileChangeOps != 0 && filepath.Clean(event.Name) == path
}

func signalReloadFileChange(changes chan<- struct{}) {
	select {
	case changes <- struct{}{}:
	default:
	}
}

func signalReloadFileError(errors chan<- error, err error) {
	select {
	case errors <- err:
	default:
	}
}

// ReloadFileSignature tracks metadata that changes when a small live config file
// is replaced or edited, without reading the file on every poll.
type ReloadFileSignature struct {
	size     int64
	mode     os.FileMode
	modTime  time.Time
	changeID string
}

func StatReloadFile(path string) (ReloadFileSignature, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ReloadFileSignature{}, err
	}
	return ReloadFileSignature{
		size:     info.Size(),
		mode:     info.Mode(),
		modTime:  info.ModTime(),
		changeID: statChangeID(info.Sys()),
	}, nil
}

func statChangeID(sys any) string {
	if sys == nil {
		return ""
	}
	v := reflect.ValueOf(sys)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Sprintf("%T", sys)
	}

	parts := make([]string, 0, 6)
	for _, name := range []string{"Dev", "Ino", "Gen", "Ctim", "Ctimespec", "Ctime", "Mtim", "Mtimespec", "Mtime"} {
		f := v.FieldByName(name)
		if f.IsValid() && f.CanInterface() {
			parts = append(parts, fmt.Sprintf("%s=%v", name, f.Interface()))
		}
	}
	return strings.Join(parts, ",")
}
