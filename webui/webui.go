package webui

import (
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cooperspencer/gickup/types"
	"github.com/goccy/go-yaml"
	"github.com/rs/zerolog/log"
)

// BackupStatus is the result of a single repo-to-destination backup attempt.
type BackupStatus string

const (
	StatusSuccess BackupStatus = "success"
	StatusFailed  BackupStatus = "failed"
)

// StatusFromInt converts the integer convention used by backup() (1=success, 0=failed).
func StatusFromInt(i int) BackupStatus {
	if i == 1 {
		return StatusSuccess
	}
	return StatusFailed
}

// BackupEntry records one repo-to-destination backup attempt.
type BackupEntry struct {
	Timestamp  time.Time    `json:"timestamp"`
	RepoName   string       `json:"repo_name"`
	RepoURL    string       `json:"repo_url"`
	Owner      string       `json:"owner"`
	Hoster     string       `json:"hoster"`
	DestType   string       `json:"dest_type"`
	DestAddr   string       `json:"dest_addr"`
	Status     BackupStatus `json:"status"`
	DurationMs int64        `json:"duration_ms"`
}

// ConfigInfo describes one loaded configuration block for the UI.
type ConfigInfo struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	Sources  int    `json:"sources"`
	Dests    int    `json:"dests"`
	CronSpec string `json:"cron_spec,omitempty"`
	NextRun  string `json:"next_run,omitempty"` // RFC3339
}

// store holds backup entries, config file paths, and the run callback.
type store struct {
	mu          sync.RWMutex
	entries     []BackupEntry
	configFiles []string
	configs     []ConfigInfo
	runFunc     func(int)
	running     int32 // atomic: 1 = a run is in progress
}

// Global is the shared store used across the application.
var Global = &store{}

// SetConfigFiles registers the config file paths with the store.
func (s *store) SetConfigFiles(files []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configFiles = make([]string, len(files))
	copy(s.configFiles, files)
}

// SetConfigs registers the list of config summaries shown in the UI.
func (s *store) SetConfigs(infos []ConfigInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs = make([]ConfigInfo, len(infos))
	copy(s.configs, infos)
}

// SetRunFunc registers the callback that main.go uses to trigger a backup run.
// index == -1 means run all configs; otherwise run the config at that index.
func (s *store) SetRunFunc(fn func(int)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runFunc = fn
}

// Record appends a backup entry to the store.
func (s *store) Record(e BackupEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
}

// Clear removes all recorded backup entries.
func (s *store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
}

//go:embed ui/index.html
var indexHTML []byte

//go:embed ui/logo.png
var logoPNG []byte

// Serve starts the web UI HTTP server on addr.
func Serve(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/logo.png", handleLogo)
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/config", handleConfig)
	mux.HandleFunc("/api/configs", handleConfigs)
	mux.HandleFunc("/api/run", handleRun)
	mux.HandleFunc("/api/running", handleRunning)
	log.Info().Str("addr", addr).Msg("Web UI listening")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error().Err(err).Msg("Web UI server error")
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

func handleLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(logoPNG)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		Global.Clear()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	Global.mu.RLock()
	entries := make([]BackupEntry, len(Global.entries))
	copy(entries, Global.entries)
	Global.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}

// handleConfigs returns the list of loaded configuration blocks.
func handleConfigs(w http.ResponseWriter, r *http.Request) {
	Global.mu.RLock()
	cfgs := make([]ConfigInfo, len(Global.configs))
	copy(cfgs, Global.configs)
	Global.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfgs)
}

// handleRunning reports whether a backup run is currently in progress.
func handleRunning(w http.ResponseWriter, r *http.Request) {
	running := atomic.LoadInt32(&Global.running) != 0
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"running": running})
}

// handleRun triggers an immediate backup run.
// POST /api/run  body: {"index": -1}   (-1 = all, ≥0 = specific config)
func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !atomic.CompareAndSwapInt32(&Global.running, 0, 1) {
		http.Error(w, "backup already running", http.StatusConflict)
		return
	}

	var req struct {
		Index int `json:"index"`
	}
	req.Index = -1
	_ = json.NewDecoder(r.Body).Decode(&req)

	Global.mu.RLock()
	fn := Global.runFunc
	Global.mu.RUnlock()

	if fn == nil {
		atomic.StoreInt32(&Global.running, 0)
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}

	go func() {
		defer atomic.StoreInt32(&Global.running, 0)
		fn(req.Index)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]bool{"running": true})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	Global.mu.RLock()
	files := make([]string, len(Global.configFiles))
	copy(files, Global.configFiles)
	Global.mu.RUnlock()

	if len(files) == 0 {
		http.Error(w, "no config files registered", http.StatusNotFound)
		return
	}

	q := r.URL.Query()

	// List mode: GET /api/config?list=1
	if r.Method == http.MethodGet && q.Get("list") == "1" {
		type fileInfo struct {
			Index int    `json:"index"`
			Name  string `json:"name"`
			Path  string `json:"path"`
		}
		list := make([]fileInfo, len(files))
		for i, f := range files {
			list[i] = fileInfo{Index: i, Name: filepath.Base(f), Path: f}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
		return
	}

	// Parse file index (default 0).
	idx := 0
	if s := q.Get("file"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 || n >= len(files) {
			http.Error(w, "invalid file index", http.StatusBadRequest)
			return
		}
		idx = n
	}

	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(files[idx])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(data)

	case http.MethodPost:
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Validate that the submitted content parses as a known config shape.
		var conf types.Conf
		if err := yaml.Unmarshal(body, &conf); err != nil {
			http.Error(w, "invalid YAML: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.WriteFile(files[idx], body, 0o600); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
