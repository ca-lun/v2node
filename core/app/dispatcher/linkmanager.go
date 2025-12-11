package dispatcher

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
)

type ManagedWriter struct {
	writer     buf.Writer
	manager    *LinkManager
	lastActive atomic.Int64 // Unix timestamp in seconds
	closed     atomic.Bool
}

func (w *ManagedWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	w.lastActive.Store(time.Now().Unix())
	return w.writer.WriteMultiBuffer(mb)
}

func (w *ManagedWriter) Close() error {
	if w.closed.Swap(true) {
		return nil // Already closed
	}
	w.manager.RemoveWriter(w)
	return common.Close(w.writer)
}

// LastActiveTime returns the last active time of the writer
func (w *ManagedWriter) LastActiveTime() time.Time {
	ts := w.lastActive.Load()
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

// IsIdle checks if the writer has been idle for the given duration
func (w *ManagedWriter) IsIdle(maxIdleTime time.Duration) bool {
	ts := w.lastActive.Load()
	if ts == 0 {
		return false // Never active, don't consider as idle
	}
	return time.Since(time.Unix(ts, 0)) > maxIdleTime
}

type LinkManager struct {
	links map[*ManagedWriter]buf.Reader
	mu    sync.RWMutex
}

func (m *LinkManager) AddLink(writer *ManagedWriter, reader buf.Reader) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.links == nil {
		m.links = make(map[*ManagedWriter]buf.Reader)
	}
	// Initialize lastActive to now
	writer.lastActive.Store(time.Now().Unix())
	m.links[writer] = reader
}

func (m *LinkManager) RemoveWriter(writer *ManagedWriter) {
	m.mu.Lock()
	r := m.links[writer]
	delete(m.links, writer)
	m.mu.Unlock()
	if r != nil {
		// Interrupt the reader to ensure reader goroutines stop
		common.Interrupt(r)
	}
}

// LinkCount returns the number of active links
func (m *LinkManager) LinkCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.links)
}

// CloseIdleLinks closes connections that have been idle for longer than maxIdleTime
// Returns the number of closed connections
func (m *LinkManager) CloseIdleLinks(maxIdleTime time.Duration) int {
	m.mu.Lock()
	// Find idle links
	var idleWriters []*ManagedWriter
	var idleReaders []buf.Reader
	for w, r := range m.links {
		if w.IsIdle(maxIdleTime) {
			idleWriters = append(idleWriters, w)
			idleReaders = append(idleReaders, r)
			delete(m.links, w)
		}
	}
	m.mu.Unlock()

	// Close idle links outside the lock
	for i, w := range idleWriters {
		if !w.closed.Swap(true) {
			common.Close(w.writer)
		}
		if idleReaders[i] != nil {
			common.Interrupt(idleReaders[i])
		}
	}

	return len(idleWriters)
}

func (m *LinkManager) CloseAll() {
	m.mu.Lock()
	// Copy links and clear map under lock to avoid races and deadlock
	links := make(map[*ManagedWriter]buf.Reader, len(m.links))
	for w, r := range m.links {
		links[w] = r
	}
	// clear original map
	m.links = make(map[*ManagedWriter]buf.Reader)
	m.mu.Unlock()

	// Close writers and interrupt readers without holding the manager lock
	for w, r := range links {
		// Close the writer. Do not call ManagedWriter.Close while holding mu to avoid reentrance.
		if !w.closed.Swap(true) {
			common.Close(w.writer)
		}
		common.Interrupt(r)
	}
}
