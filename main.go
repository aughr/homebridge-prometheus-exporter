package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	Username string
	Password string
	URI      string
	KeyFile  string
	Port     int
	Prefix   string
	Debug    bool
}

func main() {
	config := parseFlags()

	if config.Debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Debug mode enabled")
	}

	log.Printf("Starting homebridge-exporter with config: username=%s, uri=%s, port=%d, prefix=%s",
		config.Username, config.URI, config.Port, config.Prefix)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	server := startMetricsServer(ctx, &wg, config)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	log.Println("Received interrupt signal, shutting down gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	cancel()
	wg.Wait()
	log.Println("Server exited")
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.Username, "u", "", "Homebridge username")
	flag.StringVar(&config.Username, "username", "", "Homebridge username")
	flag.StringVar(&config.Password, "p", "", "Homebridge password")
	flag.StringVar(&config.Password, "password", "", "Homebridge password")
	flag.StringVar(&config.URI, "uri", "http://localhost:8581", "Homebridge UI uri")
	flag.StringVar(&config.KeyFile, "keyfile", "authorization-keys.yml", "Authorization keys file")
	flag.IntVar(&config.Port, "port", 9123, "Metrics webserver port")
	flag.StringVar(&config.Prefix, "prefix", "homebridge", "Registry metrics prefix")
	flag.BoolVar(&config.Debug, "debug", false, "Debug mode")

	flag.Parse()

	if config.Username == "" || config.Password == "" {
		fmt.Fprintf(os.Stderr, "Username and password are required\n")
		flag.Usage()
		os.Exit(1)
	}

	return config
}

func startMetricsServer(ctx context.Context, wg *sync.WaitGroup, config *Config) *http.Server {
	session := NewSession(config.Username, config.Password, config.URI)
	keys := loadKeys(config.KeyFile)

	appState := &AppState{
		session: session,
		config:  config,
		keys:    keys,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", pingHandler(appState))
	mux.Handle("/metrics", metricsHandler(appState))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: mux,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting HTTP server on port %d", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	return server
}

func pingHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("PONG"))
	}
}

func metricsHandler(state *AppState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := state.session.GetToken()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		registry, err := buildRegistry(token, state.config.URI, state.config.Prefix)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		handler.ServeHTTP(w, r)
	})
}
