package dsp

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	SUBJECTRequest   = "request"
	SUBJECTResponse  = "response"
	SUBJECTAttribute = "attribute"
	SUBJECTWinLoss   = "winloss"
)

type FileWriters struct {
	existing                              int
	request, response, attribute, winloss string
	Interval                              int
	FHRequest                             *os.File
	FHResponse                            *os.File
	FHAttribute                           *os.File
	FHWinLoss                             *os.File
}

func getCurrent(interval int) int {
	return int(time.Now().Unix() / int64(interval*60))
}

func newFileWriter(name string) (*os.File, error) {
	fh, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	return fh, nil
}

// NewFileWriters creates a new FileWriters with the given file names, and intervals in minutes
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
	fw := &FileWriters{int(existing), request, response, attribute, winloss, interval, nil, nil, nil, nil}
	return fw, nil
}

func (self *FileWriters) ReceiveLogs(nc *nats.Conn) error {
	successchan := make(chan bool)
	errchan := make(chan error)

	_, err := nc.Subscribe("*", func(m *nats.Msg) {
		current := getCurrent(self.Interval)
		if current > self.existing {
			self.existing = current
			if self.FHRequest != nil {
				self.FHRequest.Close()
				self.FHRequest = nil
			}
			if self.FHResponse != nil {
				self.FHResponse.Close()
				self.FHResponse = nil
			}
			if self.FHAttribute != nil {
				self.FHAttribute.Close()
				self.FHAttribute = nil
			}
			if self.FHWinLoss != nil {
				self.FHWinLoss.Close()
				self.FHWinLoss = nil
			}
		}
		var err error
		switch m.Subject {
		case SUBJECTRequest:
			if self.FHRequest == nil {
				self.FHRequest, err = newFileWriter(fmt.Sprintf("%s/%d", self.request, current))
				if err != nil {
					break
				}
			}
			if _, err = self.FHRequest.Write(m.Data); err == nil {
				_, err = self.FHRequest.Write([]byte("\n"))
			}
		case SUBJECTResponse:
			if self.FHResponse == nil {
				self.FHResponse, err = newFileWriter(fmt.Sprintf("%s/%d", self.response, current))
				if err != nil {
					break
				}
			}
			if _, err = self.FHResponse.Write(m.Data); err == nil {
				_, err = self.FHResponse.Write([]byte("\n"))
			}
		case SUBJECTAttribute:
			if self.FHAttribute == nil {
				self.FHAttribute, err = newFileWriter(fmt.Sprintf("%s/%d", self.attribute, current))
				if err != nil {
					break
				}
			}
			if _, err = self.FHAttribute.Write(m.Data); err == nil {
				_, err = self.FHAttribute.Write([]byte("\n"))
			}
		case SUBJECTWinLoss:
			if self.FHWinLoss == nil {
				self.FHWinLoss, err = newFileWriter(fmt.Sprintf("%s/%d", self.winloss, current))
				if err != nil {
					break
				}
			}
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
			if self.FHAttribute != nil {
				self.FHAttribute.Close()
			}
			if self.FHRequest != nil {
				self.FHRequest.Close()
			}
			if self.FHResponse != nil {
				self.FHResponse.Close()
			}
			if self.FHWinLoss != nil {
				self.FHWinLoss.Close()
			}
			return errs
		}
	}
}
