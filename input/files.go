package input

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
)

// File 用于存储文件状态
type Files struct {
	dirty   atomic.Bool
	path   	string
	logFile string
	States 	map[uint64]*FileState `json:"states"`
	ch     	chan *FileState
}


func NewFiles(registryPath, logFile string) (*Files, error) {
	r := &Files{
		States: make(map[uint64]*FileState),
		path:   registryPath,
		logFile: logFile,
		ch:      make(chan *FileState, 1),
	}
	// 如果注册表文件存在，加载它
	if _, err := os.Stat(r.path); err == nil {
		data, err := os.ReadFile(r.path)
		if err != nil {
			return nil, fmt.Errorf("failed to read registry file: %v", err)
		}
		if err := json.Unmarshal(data, &r.States); err != nil {
			return nil, fmt.Errorf("failed to unmarshal registry: %v", err)
		}
	}
	dir := filepath.Dir(r.logFile)
	base := filepath.Base(r.logFile)
	// 扫描历史日志文件
	if err := r.scanFiles(dir, base); err != nil {
		return nil, fmt.Errorf("failed to scan historical files: %v", err)
	}
	// 保存注册表
	if err := r.Save(); err != nil {
		return nil, fmt.Errorf("failed to save registry: %v", err)
	}
	if err := r.watchDirectory(dir); err != nil {
		return nil, fmt.Errorf("failed to start watching directory: %v", err)
	}
	return r, nil
}

func (r *Files) List() <-chan *FileState {
	return r.ch
}

func (r *Files) SetDirty() {
	r.dirty.Store(true)
}

func (r *Files) Save() error {
	if !r.dirty.Load() {
		return nil
	}
	data, err := json.MarshalIndent(r.States, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %v", err)
	}
	// 先写入临时文件，然后重命名，保证原子性
	tempFile := r.path + ".tmp"
	if errx := os.WriteFile(tempFile, data, 0o644); errx != nil {
		return fmt.Errorf("failed to write registry file: %v", errx)
	}
	err = os.Rename(tempFile, r.path)
	if err != nil {
		return fmt.Errorf("failed to rename registry file: %v", err)
	}
	r.dirty.Store(false)
	return nil
}

// 新增：扫描历史日志文件
func (r *Files) scanFiles(dir, base string) error {
	fileName := filepath.Base(base)
	fileName = strings.TrimSuffix(fileName, ".log")
	// 构建用于匹配日志文件的正则表达式
	pattern := fmt.Sprintf("^(%s).*\\.log$", regexp.QuoteMeta(fileName))
	logFileRegex := regexp.MustCompile(pattern)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	states := make(map[uint64]*FileState)
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !logFileRegex.MatchString(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		state, err := NewFileState(path)
		if err != nil {
			continue
		}
		states[state.Inode] = state
	}
	for inode, state := range states {
		if oldState := r.States[inode]; oldState != nil {
			state.Offset = oldState.Offset
		}
	}
	r.States = states
	r.SetDirty()
	return nil
}

func (r *Files) addFile(state *FileState) {
	if oldState := r.States[state.Inode]; oldState != nil {
		oldState.Path = state.Path
		oldState.Size = state.Size
		oldState.Created = state.Created
		oldState.Modified = state.Modified
		return
	}
	r.States[state.Inode] = state
	r.ch <- state
}

func (r *Files) Close() {
	close(r.ch)
	r.Save()
}

func (r *Files) watchDirectory(dir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %v", err)
	}
	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("failed to watch directory: %v", err)
	}
	go func() {
		defer watcher.Close()
		for _, state := range r.States {
			r.ch <- state
		}
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {
					logFile := event.Name
					if !strings.HasSuffix(logFile, ".log") {
						continue
					}
					fileState, err := NewFileState(logFile)
					if err != nil {
						log.Printf("Error getting info for file %s: %v", logFile, err)
						continue
					}
					r.addFile(fileState)
				}
			case err := <-watcher.Errors:
				log.Printf("Watcher error: %v", err)
			}
		}
	}()
	return nil
}

