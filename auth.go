package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type AuthorizationKeys struct {
	Keys []string `yaml:"keys"`
}

type AppState struct {
	session *Session
	config  *Config
	keys    *AuthorizationKeys
}

func loadKeys(keyfilePath string) *AuthorizationKeys {
	file, err := os.Open(keyfilePath)
	if err != nil {
		log.Printf("Could not open authorization key file: %v", err)
		log.Println("Using an empty key set, authorization won't be available.")
		return &AuthorizationKeys{Keys: []string{}}
	}
	defer file.Close()

	var keys AuthorizationKeys
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&keys); err != nil {
		log.Printf("Could not read values from authorization key file: %v", err)
		return &AuthorizationKeys{Keys: []string{}}
	}

	log.Printf("Loaded %d authorization keys", len(keys.Keys))
	return &keys
}

func checkBearerToken(r *http.Request, keys []string) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return false
	}

	token := parts[1]
	for _, key := range keys {
		if key == token {
			return true
		}
	}

	return false
}