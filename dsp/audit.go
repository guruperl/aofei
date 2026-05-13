package dsp

import (
	"sync"
	"sync/atomic"

	"github.com/nats-io/nats.go"
)

const defaultAuditQueueSize = 1024

type auditMessage struct {
	subject string
	data    []byte
}

type auditPublisher struct {
	nc     *nats.Conn
	queue  chan auditMessage
	mu     sync.RWMutex
	wg     sync.WaitGroup
	drops  atomic.Uint64
	closed bool
}

func newAuditPublisher(nc *nats.Conn, queueSize int) *auditPublisher {
	if queueSize <= 0 {
		queueSize = defaultAuditQueueSize
	}
	publisher := &auditPublisher{
		nc:    nc,
		queue: make(chan auditMessage, queueSize),
	}
	if nc != nil {
		publisher.wg.Add(1)
		go publisher.run()
	}
	return publisher
}

func (p *auditPublisher) Enqueue(subject string, data []byte) {
	if p == nil {
		return
	}
	copied := append([]byte(nil), data...)
	msg := auditMessage{subject: subject, data: copied}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return
	}
	select {
	case p.queue <- msg:
		metricAuditEnqueued.Add(1)
	default:
		p.drops.Add(1)
		metricAuditDropped.Add(1)
	}
}

func (p *auditPublisher) Dropped() uint64 {
	if p == nil {
		return 0
	}
	return p.drops.Load()
}

func (p *auditPublisher) Close() {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	close(p.queue)
	p.mu.Unlock()
	p.wg.Wait()
	if p.nc != nil {
		_ = p.nc.Flush()
	}
}

func (p *auditPublisher) run() {
	defer p.wg.Done()
	for msg := range p.queue {
		if p.nc == nil {
			continue
		}
		if err := p.nc.Publish(msg.subject, msg.data); err != nil {
			metricAuditPublishErrors.Add(1)
		}
	}
}
