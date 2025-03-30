package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/match"
	"github.com/nats-io/nats.go"
)

// FileWriters is a struct that contains the file handlers for writing logs
// Note that we are using *os.File instead of io.Writer because we need to
// close the file handlers in case of log rotation or in error.
type FileWriters struct {
	existing int
	top      string
}

// NewFileWriters creates a new FileWriters with the given directory names, and intervals in minutes
func NewFileWriters(top string) (*FileWriters, error) {
	if err := os.MkdirAll(top, os.ModePerm); err != nil {
		return nil, err
	}
	return &FileWriters{top: top}, nil
}

// ReceiveLogs receives logs from nats server and writes them to files
func (self *FileWriters) ReceiveLogs(nc *nats.Conn) error {
	successchan := make(chan bool)
	errchan := make(chan error)

	_, err := nc.Subscribe("*", func(m *nats.Msg) {
		switch m.Subject {
		case dsp.SUBJECTRequest:
		case dsp.SUBJECTResponse:
		case dsp.SUBJECTAttribute:
		case dsp.SUBJECTWinLoss:
		default:
		}

		filename := strings.ReplaceAll(m.Subject, ":", "/")
		dir, base := filepath.Split(filename)
		if strings.HasPrefix(dir, acl.HashNamePubmap) ||
			strings.HasPrefix(dir, match.HashNameAudience) ||
			strings.HasPrefix(dir, match.HashNameSlot) ||
			strings.HasPrefix(dir, match.HashNameCreative) {
			var err error
			if err = os.MkdirAll(fmt.Sprintf("%s/%s", self.top, dir), os.ModePerm); err == nil {
				var w *os.File
				if w, err = os.OpenFile(fmt.Sprintf("%s/%s/%s", self.top, dir, base), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666); err == nil {
					defer w.Close()
					_, err = io.Copy(w, bytes.NewReader(m.Data))
				}
			}
			if err != nil {
				errchan <- err
			}
		}
		successchan <- true
	})
	return err
}
