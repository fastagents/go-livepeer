package core

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
)

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
