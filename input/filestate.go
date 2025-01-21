package input

import (
	"io"
	"os"
	"syscall"
	"time"
)

// FileState 存储文件的处理状态
type FileState struct {
	Path     string    `json:"path"`
	Offset   int64     `json:"offset"`
	Size     int64     `json:"size"`
	Inode    uint64    `json:"inode"`
	Device   uint64    `json:"device"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`

	file *os.File
}

func NewFileState(path string) (*FileState, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	stat := info.Sys().(*syscall.Stat_t)
	state := &FileState{
		Path:     path,
		Size:	  stat.Size,
		Inode:    stat.Ino,
		Device:   stat.Dev,
		Created:  info.ModTime(),
		Modified: info.ModTime(),
	}
	return state, nil
}

func (f *FileState) Open() (*Reader, error) {
	file, err := os.Open(f.Path)
	if err != nil {
		return nil, err
	}
	file.Seek(f.Offset, io.SeekStart)
	f.file = file
	return NewReader(f, file), nil
}

func (f *FileState) Close() error {
	if f.file != nil {
		f.file.Close()
	}
	return nil
}
