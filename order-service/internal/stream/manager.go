package stream

import "sync"

type Subscriber chan string

type Manager struct {
	mu          sync.RWMutex
	subscribers map[string]map[Subscriber]struct{}
}

func NewManager() *Manager {
	return &Manager{
		subscribers: make(map[string]map[Subscriber]struct{}),
	}
}

func (m *Manager) Subscribe(orderID string) Subscriber {
	ch := make(Subscriber, 10)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.subscribers[orderID]; !ok {
		m.subscribers[orderID] = make(map[Subscriber]struct{})
	}
	m.subscribers[orderID][ch] = struct{}{}

	return ch
}

func (m *Manager) Unsubscribe(orderID string, ch Subscriber) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if subs, ok := m.subscribers[orderID]; ok {
		delete(subs, ch)
		close(ch)

		if len(subs) == 0 {
			delete(m.subscribers, orderID)
		}
	}
}

func (m *Manager) Publish(orderID, status string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subs, ok := m.subscribers[orderID]
	if !ok {
		return
	}

	for ch := range subs {
		select {
		case ch <- status:
		default:
		}
	}
}
