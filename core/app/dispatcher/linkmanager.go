package dispatcher

import (
	sync "sync"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
)

type ManagedWriter struct {
	writer  buf.Writer
	manager *LinkManager
}

func (w *ManagedWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	return w.writer.WriteMultiBuffer(mb)
}

func (w *ManagedWriter) Close() error {
	w.manager.RemoveWriter(w)
	return common.Close(w.writer)
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
		common.Close(w)
		common.Interrupt(r)
	}
}
