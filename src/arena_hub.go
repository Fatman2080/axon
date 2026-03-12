package main

import "sync"

type agentArenaHub struct {
	mu          sync.RWMutex
	snapshot    AgentArenaSnapshotResponse
	subscribers map[int]chan []AgentArenaEvent
	nextID      int
}

func newAgentArenaHub() *agentArenaHub {
	return &agentArenaHub{
		subscribers: make(map[int]chan []AgentArenaEvent),
		snapshot: AgentArenaSnapshotResponse{
			GeneratedAt: nowISO(),
			Agents:      make([]AgentArenaAgent, 0),
		},
	}
}

func (h *agentArenaHub) snapshotCopy() AgentArenaSnapshotResponse {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return cloneArenaSnapshot(h.snapshot)
}

func (h *agentArenaHub) subscribe(buffer int) (<-chan []AgentArenaEvent, func()) {
	if buffer <= 0 {
		buffer = 8
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.nextID
	h.nextID++
	ch := make(chan []AgentArenaEvent, buffer)
	h.subscribers[id] = ch

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if existing, ok := h.subscribers[id]; ok {
			delete(h.subscribers, id)
			close(existing)
		}
	}
}

func (h *agentArenaHub) subscribeWithSnapshot(buffer int) (<-chan []AgentArenaEvent, AgentArenaSnapshotResponse, func()) {
	if buffer <= 0 {
		buffer = 8
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.nextID
	h.nextID++
	ch := make(chan []AgentArenaEvent, buffer)
	h.subscribers[id] = ch
	snapshot := cloneArenaSnapshot(h.snapshot)

	return ch, snapshot, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if existing, ok := h.subscribers[id]; ok {
			delete(h.subscribers, id)
			close(existing)
		}
	}
}

func (h *agentArenaHub) reconcile(next AgentArenaSnapshotResponse) []AgentArenaEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	events := diffAgentArenaSnapshots(h.snapshot, next)
	h.snapshot = cloneArenaSnapshot(next)
	if len(events) == 0 {
		return nil
	}

	for id, subscriber := range h.subscribers {
		payload := cloneArenaEvents(events)
		select {
		case subscriber <- payload:
		default:
			close(subscriber)
			delete(h.subscribers, id)
		}
	}
	return events
}

func cloneArenaSnapshot(snapshot AgentArenaSnapshotResponse) AgentArenaSnapshotResponse {
	cloned := snapshot
	if len(snapshot.Agents) == 0 {
		cloned.Agents = make([]AgentArenaAgent, 0)
		return cloned
	}

	cloned.Agents = make([]AgentArenaAgent, len(snapshot.Agents))
	copy(cloned.Agents, snapshot.Agents)
	return cloned
}

func cloneArenaEvents(events []AgentArenaEvent) []AgentArenaEvent {
	cloned := make([]AgentArenaEvent, len(events))
	for index, event := range events {
		cloned[index] = event
		if event.Agent != nil {
			agentCopy := *event.Agent
			cloned[index].Agent = &agentCopy
		}
	}
	return cloned
}
