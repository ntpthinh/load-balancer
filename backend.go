package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

type Backend interface {
	IsAlive() bool
	SetAlive(bool)
	Serve(http.ResponseWriter, *http.Request)
	GetUrl() string
}

func (b *backend) IsAlive() (alive bool) {
	b.mux.RLock()
	alive = b.Alive
	b.mux.RUnlock()
	return
}

func (b *backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *backend) Serve(w http.ResponseWriter, r *http.Request) {
	b.ReverseProxy.ServeHTTP(w, r)
}

func (b *backend) GetUrl() string{
	return b.URL.String()
}