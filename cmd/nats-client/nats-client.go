package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/genelet/winter/dsp"
	"github.com/nats-io/nats.go"
)

const logQueueSize = 1024

var ErrIgnoredSubject = errors.New("ignored nats log subject")

type logMessage struct {
	subject string
	data    []byte
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
}

func getCurrent(interval int) int {
	return int(time.Now().Unix() / int64(interval*60))
}

func newFileWriter(name string) (*os.File, error) {
	fh, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return fh, nil
}

// NewFileWriters creates a new FileWriters with the given directory names, and intervals in minutes
func NewFileWriters(request, response, attribute, winloss string, interval int) (*FileWriters, error) {
	existing := getCurrent(interval)
	if err := os.MkdirAll(request, os.ModePerm); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(response, os.ModePerm); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(attribute, os.ModePerm); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(winloss, os.ModePerm); err != nil {
		return nil, err
	}
	fw := &FileWriters{existing: int(existing), request: request, response: response, attribute: attribute, winloss: winloss, Interval: interval, current: getCurrent}
	return fw, nil
}

// ReceiveLogs receives logs from nats server and writes them to files
func (self *FileWriters) ReceiveLogs(nc *nats.Conn) error {
	msgs := make(chan logMessage, logQueueSize)
	errs := make(chan error, 1)

	_, err := nc.Subscribe("*", func(m *nats.Msg) {
		enqueueLogMessage(msgs, errs, m.Subject, m.Data)
	})
	if err != nil {
		return err
	}

	return self.writerLoop(msgs, errs)
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

func (self *FileWriters) writerLoop(msgs <-chan logMessage, errs <-chan error) error {
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
