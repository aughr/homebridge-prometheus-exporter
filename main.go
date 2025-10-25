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

	"github.com/prometheus/client_golang/prometheus"
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

	// Get defaults from environment variables
	envUsername := os.Getenv("HOMEBRIDGE_USERNAME")
	envPassword := os.Getenv("HOMEBRIDGE_PASSWORD")
	envURI := getEnvOrDefault("HOMEBRIDGE_URI", "http://localhost:8581")
	envKeyFile := getEnvOrDefault("HOMEBRIDGE_KEYFILE", "authorization-keys.yml")
	envPort := getEnvIntOrDefault("HOMEBRIDGE_PORT", 9123)
	envPrefix := getEnvOrDefault("HOMEBRIDGE_PREFIX", "homebridge")
	envDebug := getEnvBoolOrDefault("HOMEBRIDGE_DEBUG", false)

	flag.StringVar(&config.Username, "u", envUsername, "Homebridge username (env: HOMEBRIDGE_USERNAME)")
	flag.StringVar(&config.Username, "username", envUsername, "Homebridge username (env: HOMEBRIDGE_USERNAME)")
	flag.StringVar(&config.Password, "p", envPassword, "Homebridge password (env: HOMEBRIDGE_PASSWORD)")
	flag.StringVar(&config.Password, "password", envPassword, "Homebridge password (env: HOMEBRIDGE_PASSWORD)")
	flag.StringVar(&config.URI, "uri", envURI, "Homebridge UI uri (env: HOMEBRIDGE_URI)")
	flag.StringVar(&config.KeyFile, "keyfile", envKeyFile, "Authorization keys file (env: HOMEBRIDGE_KEYFILE)")
	flag.IntVar(&config.Port, "port", envPort, "Metrics webserver port (env: HOMEBRIDGE_PORT)")
	flag.StringVar(&config.Prefix, "prefix", envPrefix, "Registry metrics prefix (env: HOMEBRIDGE_PREFIX)")
	flag.BoolVar(&config.Debug, "debug", envDebug, "Debug mode (env: HOMEBRIDGE_DEBUG)")

	flag.Parse()

	if config.Username == "" || config.Password == "" {
		fmt.Fprintf(os.Stderr, "Username and password are required\n")
		flag.Usage()
		os.Exit(1)
	}

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := parseInt(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func startMetricsServer(ctx context.Context, wg *sync.WaitGroup, config *Config) *http.Server {
	session := NewSession(config.Username, config.Password, config.URI, config.Debug)
	keys := loadKeys(config.KeyFile)

	registry := prometheus.NewRegistry()
	collector := NewHomebridgeCollector(session, config)
	registry.MustRegister(collector)

	appState := &AppState{
		session:  session,
		config:   config,
		keys:     keys,
		registry: registry,
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
	handler := promhttp.HandlerFor(state.registry, promhttp.HandlerOpts{})
	return handler
}
