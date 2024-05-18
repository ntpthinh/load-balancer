package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
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
	HealthCheck()
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
func isBackendAlive(url *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", url.Host, timeout)
	if err != nil {
		log.Println("Site unreachable, err:", err)
		return false
	}
	conn.Close()
	return true
}
func (s *roundRobinServerPool) HealthCheck() {
	for _, b := range s.servers {
		alive := isBackendAlive(b.GetUrl())
		b.SetAlive(alive)
	}
}
