package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/go-redis/redis/v8" // Or use etcd if preferred
	"github.com/joho/godotenv"
)

var (
	proxyConfig = struct {
		sync.RWMutex
		routes map[string]*httputil.ReverseProxy
	}{
		routes: make(map[string]*httputil.ReverseProxy),
	}

	ctx = context.Background()
)

func main() {
	env, err := godotenv.Read()
	if err != nil {
		log.Println("Error loading .env file:", err)
	}

	// Set up Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     env["REDIS_HOST"],
		Password: env["REDIS_PASSWORD"],
	})

	// Initial route fetch from Redis
	fetchRoutes(rdb)

	// Periodically update the proxy configuration
	go func() {
		for {
			time.Sleep(10 * time.Second)
			fetchRoutes(rdb)
		}
	}()

	// Set up the reverse proxy handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxyConfig.RLock()
		proxy, ok := proxyConfig.routes[r.Host]
		proxyConfig.RUnlock()
		if !ok {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	log.Println("Reverse proxy is running on :80")
	log.Fatal(http.ListenAndServe(":80", nil))
}

// Fetch routes from Redis and update the proxy configuration
func fetchRoutes(rdb *redis.Client) {
	proxyConfig.Lock()
	defer proxyConfig.Unlock()

	routes, err := rdb.HGetAll(ctx, "reverse-proxy-routes").Result()
	if err != nil {
		log.Println("Failed to fetch routes:", err)
		return
	}

	for host, backendURL := range routes {
		backend, err := url.Parse(backendURL)
		if err != nil {
			log.Printf("Invalid backend URL for %s: %s", host, err)
			continue
		}

		backend.Port()
		proxyConfig.routes[host] = httputil.NewSingleHostReverseProxy(backend)
	}

	log.Println("Routes updated")
}
