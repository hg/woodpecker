package file

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	"go.woodpecker-ci.org/woodpecker/v2/server/services/log"
)

type logFiles struct {
	sync.Mutex
	byStepId map[int64]*os.File
}

func (f *logFiles) getFile(stepId int64) *os.File {
	f.Lock()
	file := f.byStepId[stepId]
	f.Unlock()
	return file
}

func (f *logFiles) setFile(stepId int64, file *os.File) {
	f.Lock()
	f.byStepId[stepId] = file
	f.Unlock()
}

func (f *logFiles) closeFile(stepId int64) error {
	f.Lock()
	defer f.Unlock()

	file := f.byStepId[stepId]
	if file == nil {
		return nil
	}

	delete(f.byStepId, stepId)
	return file.Close()
}

type logStore struct {
	base  string
	files *logFiles
}

func NewLogStore(base string) (log.Service, error) {
	if base == "" {
		return nil, fmt.Errorf("file storage base path is required")
	}
	if _, err := os.Stat(base); err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(base, 0o600)
		if err != nil {
			return nil, err
		}
	}
	files := &logFiles{
		Mutex:    sync.Mutex{},
		byStepId: make(map[int64]*os.File),
	}
	return logStore{base: base, files: files}, nil
}

func (l logStore) filePath(id int64) string {
	return filepath.Join(l.base, fmt.Sprintf("%d.json", id))
}

func (l logStore) LogFind(step *model.Step) ([]*model.LogEntry, error) {
	filename := l.filePath(step.ID)
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	s := bufio.NewScanner(file)
	var entries []*model.LogEntry
	for s.Scan() {
		j := s.Text()
		if len(strings.TrimSpace(j)) == 0 {
			continue
		}
		entry := &model.LogEntry{}
		err = json.Unmarshal([]byte(j), entry)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (l logStore) LogAppend(logEntry *model.LogEntry) error {
	file := l.files.getFile(logEntry.StepID)

	if file == nil {
		var err error
		file, err = os.OpenFile(l.filePath(logEntry.StepID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		l.files.setFile(logEntry.StepID, file)
	}

	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		return err
	}
	_, err = file.Write(append(jsonData, byte('\n')))
	return err
}

func (l logStore) LogDelete(step *model.Step) error {
	return os.Remove(l.filePath(step.ID))
}

func (l logStore) LogFinish(step *model.Step) error {
	return l.files.closeFile(step.ID)
}
