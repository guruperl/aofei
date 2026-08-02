package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guruperl/aofei/dsp"
	"github.com/nats-io/nats.go"
)

const (
	logQueueSize = 1024
	logDirMode   = 0750
	logFileMode  = 0640
)

var ErrIgnoredSubject = errors.New("ignored nats log subject")

type logMessage struct {
	subject string
	data    []byte
}

type natsLogConn interface {
	Subscribe(string, nats.MsgHandler) (*nats.Subscription, error)
	Drain() error
	ConnectedUrl() string
}

// FileWriters is a struct that contains the file handlers for writing logs
// Note that we are using *os.File instead of io.Writer because we need to
// close the file handlers in case of log rotation or in error.
type FileWriters struct {
	existing                              int
	request, response, attribute, winloss string // directory names
	Interval                              int
	FHRequest                             *os.File
	FHResponse                            *os.File
	FHAttribute                           *os.File
	FHWinLoss                             *os.File
	current                               func(int) int
	now                                   func() time.Time
	retention                             time.Duration
}

func getCurrent(interval int) int {
	return int(time.Now().Unix() / int64(interval*60))
}

func newFileWriter(name string) (*os.File, error) {
	fh, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY|os.O_CREATE, logFileMode)
	if err != nil {
		return nil, err
	}
	if err := fh.Chmod(logFileMode); err != nil {
		_ = fh.Close()
		return nil, err
	}
	return fh, nil
}

// NewFileWriters creates a new FileWriters with the given directory names, and intervals in minutes
func NewFileWriters(request, response, attribute, winloss string, interval int) (*FileWriters, error) {
	return newFileWritersWithRetention(request, response, attribute, winloss, interval, 7*24*time.Hour, time.Now)
}

func newFileWritersWithRetention(request, response, attribute, winloss string, interval int, retention time.Duration, now func() time.Time) (*FileWriters, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("log rotation interval must be positive")
	}
	if retention <= 0 {
		return nil, fmt.Errorf("log retention must be positive")
	}
	if now == nil {
		now = time.Now
	}
	existing := getCurrent(interval)
	if err := ensureLogDir(request); err != nil {
		return nil, err
	}
	if err := ensureLogDir(response); err != nil {
		return nil, err
	}
	if err := ensureLogDir(attribute); err != nil {
		return nil, err
	}
	if err := ensureLogDir(winloss); err != nil {
		return nil, err
	}
	fw := &FileWriters{existing: int(existing), request: request, response: response, attribute: attribute, winloss: winloss, Interval: interval, current: getCurrent, now: now, retention: retention}
	if _, err := fw.pruneExpiredLogs(); err != nil {
		return nil, err
	}
	return fw, nil
}

func ensureLogDir(name string) error {
	if err := os.MkdirAll(name, logDirMode); err != nil {
		return err
	}
	return os.Chmod(name, logDirMode)
}

type logPrintf interface {
	Printf(string, ...any)
}

func runNATSClient(ctx context.Context, c *dsp.Config, nc natsLogConn, interval int, logger logPrintf) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	if c == nil {
		return fmt.Errorf("DSP config is nil")
	}
	filewriters, err := newFileWritersWithRetention(c.LogRequest, c.LogResponse, c.LogAttribute, c.LogWinLoss, interval, c.PrivacyLogRetention(), time.Now)
	if err != nil {
		return err
	}
	logger.Printf("Listening on [%s]", nc.ConnectedUrl())
	return filewriters.ReceiveLogs(ctx, nc)
}

// ReceiveLogs receives logs from nats server and writes them to files
func (self *FileWriters) ReceiveLogs(ctx context.Context, nc natsLogConn) error {
	if ctx == nil {
		ctx = context.Background()
	}
	msgs := make(chan logMessage, logQueueSize)
	errs := make(chan error, 1)

	_, err := nc.Subscribe("*", func(m *nats.Msg) {
		enqueueLogMessage(msgs, errs, m.Subject, m.Data)
	})
	if err != nil {
		return err
	}

	return self.writerLoop(ctx, nc, msgs, errs)
}

func enqueueLogMessage(msgs chan<- logMessage, errs chan<- error, subject string, data []byte) {
	copied := append([]byte(nil), data...)
	select {
	case msgs <- logMessage{subject: subject, data: copied}:
	default:
		select {
		case errs <- fmt.Errorf("nats log queue full for subject %q", subject):
		default:
		}
	}
}

func (self *FileWriters) writerLoop(ctx context.Context, nc natsLogConn, msgs <-chan logMessage, errs <-chan error) error {
	defer self.Close()
	for {
		select {
		case msg := <-msgs:
			if handled, err := self.writeLog(msg); err != nil {
				return err
			} else if !handled {
				log.Printf("%v: %s", ErrIgnoredSubject, msg.subject)
			}
		case err := <-errs:
			return err
		case <-ctx.Done():
			if err := nc.Drain(); err != nil {
				return err
			}
			return self.drainQueuedLogs(msgs, errs)
		}
	}
}

func (self *FileWriters) drainQueuedLogs(msgs <-chan logMessage, errs <-chan error) error {
	for {
		select {
		case msg := <-msgs:
			if handled, err := self.writeLog(msg); err != nil {
				return err
			} else if !handled {
				log.Printf("%v: %s", ErrIgnoredSubject, msg.subject)
			}
		case err := <-errs:
			return err
		default:
			return nil
		}
	}
}

func (self *FileWriters) WriteLog(subject string, data []byte) error {
	handled, err := self.writeLog(logMessage{subject: subject, data: data})
	if err != nil {
		return err
	}
	if !handled {
		log.Printf("%v: %s", ErrIgnoredSubject, subject)
		return fmt.Errorf("%w: %s", ErrIgnoredSubject, subject)
	}
	return nil
}

func (self *FileWriters) writeLog(msg logMessage) (bool, error) {
	current := self.current(self.Interval)
	if current > self.existing {
		self.existing = current
		self.closeFiles()
		if _, err := self.pruneExpiredLogs(); err != nil {
			return false, err
		}
	}

	var fh **os.File
	var dir string
	switch msg.subject {
	case dsp.SUBJECTRequest:
		fh = &self.FHRequest
		dir = self.request
	case dsp.SUBJECTResponse:
		fh = &self.FHResponse
		dir = self.response
	case dsp.SUBJECTAttribute:
		fh = &self.FHAttribute
		dir = self.attribute
	case dsp.SUBJECTWinLoss:
		fh = &self.FHWinLoss
		dir = self.winloss
	default:
		return false, nil
	}

	if *fh == nil {
		file, err := newFileWriter(filepath.Join(dir, fmt.Sprintf("%s.%d", msg.subject, current)))
		if err != nil {
			return true, err
		}
		*fh = file
	}
	if _, err := (*fh).Write(msg.data); err != nil {
		return true, err
	}
	if _, err := (*fh).Write([]byte("\n")); err != nil {
		return true, err
	}
	return true, nil
}

func (self *FileWriters) pruneExpiredLogs() (int, error) {
	if self == nil || self.retention <= 0 {
		return 0, fmt.Errorf("log retention is not configured")
	}
	now := time.Now
	if self.now != nil {
		now = self.now
	}
	cutoff := now().Add(-self.retention)
	targets := []struct {
		dir    string
		prefix string
	}{
		{dir: self.request, prefix: dsp.SUBJECTRequest + "."},
		{dir: self.response, prefix: dsp.SUBJECTResponse + "."},
		{dir: self.attribute, prefix: dsp.SUBJECTAttribute + "."},
		{dir: self.winloss, prefix: dsp.SUBJECTWinLoss + "."},
	}
	deleted := 0
	for _, target := range targets {
		entries, err := os.ReadDir(target.dir)
		if err != nil {
			return deleted, err
		}
		for _, entry := range entries {
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasPrefix(entry.Name(), target.prefix) {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				return deleted, err
			}
			if !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
				continue
			}
			if err := os.Remove(filepath.Join(target.dir, entry.Name())); err != nil {
				return deleted, err
			}
			deleted++
		}
	}
	return deleted, nil
}

func (self *FileWriters) Close() {
	self.closeFiles()
}

func (self *FileWriters) closeFiles() {
	if self.FHAttribute != nil {
		self.FHAttribute.Close()
		self.FHAttribute = nil
	}
	if self.FHRequest != nil {
		self.FHRequest.Close()
		self.FHRequest = nil
	}
	if self.FHResponse != nil {
		self.FHResponse.Close()
		self.FHResponse = nil
	}
	if self.FHWinLoss != nil {
		self.FHWinLoss.Close()
		self.FHWinLoss = nil
	}
}
