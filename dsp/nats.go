package dsp

import (
	"io"
	"log"
	"os"

	"github.com/nats-io/nats.go"
)

const (
	SUBJECTRequest   = "request"
	SUBJECTResponse  = "response"
	SUBJECTAttribute = "attribute"
	SUBJECTWinLoss   = "winloss"
)

type FileWriters struct {
	FHRequest   io.Writer
	FHResponse  io.Writer
	FHAttribute io.Writer
	FHWinLoss   io.Writer
}

// NewFileWriters creates a new FileWriters with the given file names.
func NewFileWriters(request, response, attribute, winloss string) (*FileWriters, error) {
	newFileWriter := func(name string) (io.Writer, error) {
		fh, err := os.Create(name)
		if err != nil {
			return nil, err
		}
		return fh, nil
	}
	fw := &FileWriters{}
	var err error
	if fw.FHRequest, err = newFileWriter(request); err != nil {
		return nil, err
	}
	if fw.FHResponse, err = newFileWriter(response); err != nil {
		return nil, err
	}
	if fw.FHAttribute, err = newFileWriter(attribute); err != nil {
		return nil, err
	}
	if fw.FHWinLoss, err = newFileWriter(winloss); err != nil {
		return nil, err
	}
	return fw, nil
}

func (self *FileWriters) ReceiveLogs(nc *nats.Conn) error {
	successchan := make(chan bool)
	errchan := make(chan error)

	_, err := nc.Subscribe("*", func(m *nats.Msg) {
		var err error
		switch m.Subject {
		case SUBJECTRequest:
			if _, err = self.FHRequest.Write(m.Data); err == nil {
				_, err = self.FHRequest.Write([]byte("\n"))
			}
		case SUBJECTResponse:
			if _, err = self.FHResponse.Write(m.Data); err == nil {
				_, err = self.FHResponse.Write([]byte("\n"))
			}
		case SUBJECTAttribute:
			if _, err = self.FHAttribute.Write(m.Data); err == nil {
				_, err = self.FHAttribute.Write([]byte("\n"))
			}
		case SUBJECTWinLoss:
			if _, err = self.FHWinLoss.Write(m.Data); err == nil {
				_, err = self.FHWinLoss.Write([]byte("\n"))
			}
		default:
		}
		if err != nil {
			errchan <- err
		}
		successchan <- true
	})
	if err != nil {
		return err
	}

	for {
		select {
		case <-successchan:
		case errs := <-errchan:
			log.Println(errs)
		}
	}
}
