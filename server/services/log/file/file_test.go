package file

import (
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	"os"
	"testing"
	"time"
)

func BenchmarkLogAppend(b *testing.B) {
	dir, err := os.MkdirTemp("", "woodpecker-test-*")
	if err != nil {
		b.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewLogStore(dir)
	if err != nil {
		b.Fatalf("error creating log store: %v", err)
	}

	data := []byte("log line text content")
	ts := time.Now().Unix()

	for i := 0; i < b.N; i++ {
		err = store.LogAppend(&model.LogEntry{
			ID:      int64(i),
			StepID:  1,
			Time:    ts,
			Line:    i,
			Data:    data,
			Created: ts,
			Type:    model.LogEntryStdout,
		})
		if err != nil {
			b.Errorf("could not append log: %v", err)
		}
	}
}
