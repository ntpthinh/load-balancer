package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

var serverPool ServerPool

var (
	Attemps int
	Retries int
)

const (
	MAX_ATTEMPTS = 3
	MAX_RETRIES  = 3
)

func main() {
	var serverList string
	var port int
	flag.StringVar(&serverList, "backends", "", "Load balanced backends, use comma to separate")
	flag.IntVar(&port, "port", 3030, "Port to serve")
	if len(serverList) == 0 {
		log.Fatal("Please provide backend to load balance")
	}

	serverPool := NewRoundRobinServerPool()
	tokens := strings.Split(serverList, ",")
	for _, token := range tokens {
		serverUrl, err := url.Parse(token)
		if err != nil {
			log.Fatal(err)
		}
		proxy := httputil.NewSingleHostReverseProxy(serverUrl)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
			log.Printf("[%s] %s \n", serverUrl.Host, e.Error())
			retries := GetRetriesFromContext(r)
			if retries < MAX_RETRIES {
				select {
				case <-time.After(10 * time.Millisecond):
					ctx := context.WithValue(r.Context(), Retries, retries+1)
					proxy.ServeHTTP(w, r.WithContext(ctx))
				}
				return
			}
			serverPool.MarkBackendStatus(serverUrl, false)
			attemps := GetAttemptsFromContext(r)
			log.Printf("%s(%s) Attempting retry %d", r.RemoteAddr, r.URL.Path, attemps+1)
			ctx := context.WithValue(r.Context(), Attemps, attemps+1)
			lb(w, r.WithContext(ctx))
		}

		serverPool.AddBackend(&backend{
			URL:          serverUrl,
			Alive:        true,
			ReverseProxy: proxy,
		})
	}
	server := http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(lb),
	}
	go healthcheck()
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
func healthcheck() {
	t := time.NewTicker(time.Minute * 2)
	for {
		select {
		case <-t.C:
			log.Println("Start health check")
			serverPool.HealthCheck()
			log.Println("Health check completed")
		}
	}
}
func GetAttemptsFromContext(r *http.Request) int {
	if attemps, ok := r.Context().Value(Attemps).(int); ok {
		return attemps
	}
	return 1
}
func GetRetriesFromContext(r *http.Request) int {
	if retries, ok := r.Context().Value(Retries).(int); ok {
		return retries
	}
	return 1
}
func lb(w http.ResponseWriter, r *http.Request) {
	attemps := GetAttemptsFromContext(r)
	if attemps > MAX_ATTEMPTS {
		log.Printf("%s(%s) Max attempts reached, terminating...", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	peer := serverPool.GetNextPeer()
	if peer != nil {
		peer.Serve(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}
