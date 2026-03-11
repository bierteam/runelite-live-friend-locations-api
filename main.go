package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	mu   sync.RWMutex
	data = make(map[string]FriendLocation)
)

type Waypoint struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Plane int `json:"plane"`
}

type PostBody struct {
	Name     string    `json:"name"`
	Waypoint *Waypoint `json:"waypoint,omitempty"`
	X        *int      `json:"x,omitempty"`
	Y        *int      `json:"y,omitempty"`
	Plane    *int      `json:"plane,omitempty"`
	Type     *string   `json:"type,omitempty"`
	Title    *string   `json:"title,omitempty"`
	World    *int      `json:"world,omitempty"`
}

type FriendLocation struct {
	Name      string  `json:"name"`
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Plane     int     `json:"plane"`
	Type      *string `json:"type,omitempty"`
	Title     *string `json:"title,omitempty"`
	World     *int    `json:"world,omitempty"`
	Timestamp int64   `json:"timestamp"`
}

const (
	sharedKeyEnv = "SHARED_KEY"
	// expiration duration for stored locations
	expirationMs    = 60 * 1000
	cleanupInterval = 10 * time.Second
)

func main() {
	sharedKey := os.Getenv(sharedKeyEnv)
	if strings.TrimSpace(sharedKey) == "" {
		log.Fatalf("%s must be set", sharedKeyEnv)
	}

	go startCleanupLoop(cleanupInterval, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", authMiddleware(sharedKey, http.HandlerFunc(handleGet)))
	mux.Handle("/post", authMiddleware(sharedKey, http.HandlerFunc(handlePost)))

	log.Print("Friend tracker listening on port 3000")
	if err := http.ListenAndServe(":3000", mux); err != nil {
		log.Fatal(err)
	}
}

func authMiddleware(sharedKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
			ip = strings.Split(xf, ",")[0]
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			log.Printf("%s No credentials sent", ip)
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "No credentials sent"})
			return
		}

		if subtle.ConstantTimeCompare([]byte(auth), []byte(sharedKey)) != 1 {
			log.Printf("%s Wrong credentials", ip)
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "Wrong credentials"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	respondJSON(w, http.StatusOK, getData())
	logRequest(r, "Data requested")
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Limit request size to prevent abuse.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var body PostBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	x, y, plane := 0, 0, 0
	if body.Waypoint != nil {
		x = body.Waypoint.X
		y = body.Waypoint.Y
		plane = body.Waypoint.Plane
	} else {
		if body.X != nil {
			x = *body.X
		}
		if body.Y != nil {
			y = *body.Y
		}
		if body.Plane != nil {
			plane = *body.Plane
		}
	}

	timestamp := time.Now().UnixMilli()
	loc := FriendLocation{
		Name:      name,
		X:         x,
		Y:         y,
		Plane:     plane,
		Type:      body.Type,
		Title:     body.Title,
		World:     body.World,
		Timestamp: timestamp,
	}

	updateData(loc)
	respondJSON(w, http.StatusOK, getData())
	logRequest(r, "Data received: %s", sanitizeBody(loc))
}

func updateData(newObj FriendLocation) {
	mu.Lock()
	defer mu.Unlock()

	data[newObj.Name] = newObj
}

func startCleanupLoop(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cleanupOld(time.Now().UnixMilli())
		case <-stop:
			return
		}
	}
}

func cleanupOld(timestamp int64) {
	mu.Lock()
	defer mu.Unlock()

	for k, v := range data {
		if timestamp-v.Timestamp > expirationMs {
			delete(data, k)
		}
	}
}

func getData() []FriendLocation {
	mu.RLock()
	defer mu.RUnlock()

	out := make([]FriendLocation, 0, len(data))
	for _, loc := range data {
		out = append(out, loc)
	}
	return out
}

func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode json response: %v", err)
	}
}

func logRequest(r *http.Request, format string, args ...interface{}) {
	ip := r.RemoteAddr
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		ip = strings.Split(xf, ",")[0]
	}
	log.Printf("%s %s", ip, fmt.Sprintf(format, args...))
}

func sanitizeBody(body FriendLocation) string {
	b, err := json.Marshal(body)
	if err != nil {
		return "<unmarshalable>"
	}
	return string(b)
}
