package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type Session struct {
	mu        sync.Mutex
	token     string
	username  string
	password  string
	uri       string
	expiresIn int64
	createdAt time.Time
	debug     bool
}

func NewSession(username, password, uri string, debug bool) *Session {
	return &Session{
		username: username,
		password: password,
		uri:      uri,
		debug:    debug,
	}
}

func (s *Session) isValid() bool {
	if s.token == "" {
		return false
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/auth/check", s.uri), nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (s *Session) GetToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isValid() {
		log.Println("Token is invalid, fetching a new token")
		token, err := login(s.username, s.password, s.uri, s.debug)
		if err != nil {
			s.token = ""
			return "", err
		}
		s.token = token.AccessToken
		s.expiresIn = token.ExpiresIn
		s.createdAt = time.Now()
	}
	return s.token, nil
}

type ServiceCharacteristics struct {
	AID         int         `json:"aid"`
	IID         int         `json:"iid"`
	UUID        string      `json:"uuid"`
	Type        string      `json:"type"`
	ServiceType string      `json:"serviceType"`
	ServiceName string      `json:"serviceName"`
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
	Format      string      `json:"format"`
	Perms       []string    `json:"perms"`
	CanRead     bool        `json:"canRead"`
	CanWrite    bool        `json:"canWrite"`
	Ev          bool        `json:"ev"`
}

type Instance struct {
	Name                  string        `json:"name"`
	Username              string        `json:"username"`
	IPAddress             string        `json:"ipAddress"`
	Port                  int           `json:"port"`
	Services              []interface{} `json:"services"`
	ConnectionFailedCount int           `json:"connectionFailedCount"`
}

type Accessory struct {
	AID                    int                      `json:"aid"`
	IID                    int                      `json:"iid"`
	UUID                   string                   `json:"uuid"`
	AccessoryType          string                   `json:"type"`
	HumanType              string                   `json:"humanType"`
	ServiceName            string                   `json:"serviceName"`
	ServiceCharacteristics []ServiceCharacteristics `json:"serviceCharacteristics"`
	AccessoryInformation   interface{}              `json:"accessoryInformation"`
	Instance               Instance                 `json:"instance"`
	Values                 interface{}              `json:"values"`
	UniqueID               string                   `json:"uniqueId"`
}

func login(username, password, uri string, debug bool) (*Token, error) {
	loginData := map[string]string{
		"username": username,
		"password": password,
		"otp":      "123",
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login data: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf("%s/api/auth/login", uri),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to make login request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("login failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %v", err)
	}

	if debug {
		log.Printf("Fetched token %s. New token is valid for %d seconds", token.AccessToken, token.ExpiresIn)
	}
	return &token, nil
}

func getAllAccessories(token, uri string, debug bool) ([]Accessory, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/accessories", uri), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	if debug {
		log.Printf("Fetching accessories using token %s", token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make accessories request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch accessories with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if debug {
		log.Printf("Accessories JSON: %s", string(body))
	}

	var accessories []Accessory
	if err := json.Unmarshal(body, &accessories); err != nil {
		return nil, fmt.Errorf("failed to unmarshal accessories: %v", err)
	}

	log.Printf("Fetched %d accessories", len(accessories))
	return accessories, nil
}
