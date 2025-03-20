package dsp

import (
	"io"
	"log"
	"os"
	"sync"

	"github.com/genelet/winter/match"
	"github.com/nats-io/nats.go"
)

const (
	SUBJECTRequest   = "request"
	SUBJECTResponse  = "response"
	SUBJECTAttribute = "attribute"
	SUBJECTWinLoss   = "winloss"
)

type AttributePlus struct {
	match.Attribute
	Adv match.RAdv
}

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

func (self *FileWriters) ReceiveLogs(nc *nats.Conn) {
	wg := sync.WaitGroup{}

	wg.Add(1)
	if _, err := nc.Subscribe(SUBJECTAttribute, func(m *nats.Msg) {
		defer wg.Done()
		self.FHAttribute.Write(m.Data)
	}); err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	if _, err := nc.Subscribe(SUBJECTWinLoss, func(m *nats.Msg) {
		defer wg.Done()
		self.FHWinLoss.Write(m.Data)
	}); err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	if _, err := nc.Subscribe(SUBJECTRequest, func(m *nats.Msg) {
		defer wg.Done()
		self.FHRequest.Write(m.Data)
	}); err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	if _, err := nc.Subscribe(SUBJECTResponse, func(m *nats.Msg) {
		defer wg.Done()
		self.FHResponse.Write(m.Data)
	}); err != nil {
		log.Fatal(err)
	}

	wg.Wait()
}
