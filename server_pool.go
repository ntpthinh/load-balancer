package main

import (
	"net/http"
	"net/url"
	"sync/atomic"
)

type roundRobinServerPool struct {
	servers []Backend
	current uint64
}

type ServerPool interface {
	GetNextPeer() Backend
	GetBackends() []Backend
	AddBackend(Backend)
	GetServerPoolSize() int
	Serve(http.ResponseWriter, *http.Request)
	MarkBackendStatus(*url.URL, bool)
}

func (s *roundRobinServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, 1) % uint64(len(s.servers)))
}

func (s *roundRobinServerPool) GetNextPeer() Backend {
	next := s.NextIndex()
	l := next + len(s.servers)
	for i := next; i < l; i++ {
		index := i % len(s.servers)
		if s.servers[index].IsAlive() {
			atomic.StoreUint64(&s.current, uint64(index))
			return s.servers[index]
		}

	}
	return nil
}

func (s *roundRobinServerPool) Serve(w http.ResponseWriter, r *http.Request) {
	s.servers[s.current].Serve(w, r)
}

func (s *roundRobinServerPool) MarkBackendStatus(backendUrl *url.URL, alive bool) {
	for _, item := range s.servers {
		if item.GetUrl() == backendUrl.String() {
			item.SetAlive(alive)
			break
		}
	}
}
