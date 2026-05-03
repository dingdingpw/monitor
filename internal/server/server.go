package server

import (
	"compress/gzip"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"vps-agent/internal/agent"
)

type Config struct {
	Addr        string
	AuthSecret  string
	AdminUser   string
	AdminPass   string
	DataPath    string
	PublicURL   string
	OfflineWait time.Duration
	MaxNodes    int
}

type Server struct {
	cfg      Config
	store    *Store
	http     *http.Server
	sessions *SessionStore
	cache    *ResponseCache
}

func New(cfg Config) (*Server, error) {
	if isWeakSecret(cfg.AuthSecret) {
		return nil, errors.New("AUTH_SECRET must not be empty or change-me")
	}
	if cfg.AdminUser == "" {
		cfg.AdminUser = "admin"
	}
	if isWeakSecret(cfg.AdminPass) {
		return nil, errors.New("ADMIN_PASS must not be empty or change-me")
	}
	if cfg.PublicURL != "" {
		publicURL, err := cleanPublicURL(cfg.PublicURL)
		if err != nil {
			return nil, err
		}
		cfg.PublicURL = publicURL
	}
	if cfg.DataPath == "" {
		cfg.DataPath = "data/server.json"
	}
	if cfg.OfflineWait == 0 {
		cfg.OfflineWait = 10 * time.Second
	}
	if cfg.MaxNodes == 0 {
		cfg.MaxNodes = 2000
	}
	store, err := NewStore(cfg.DataPath)
	if err != nil {
		return nil, err
	}
	s := &Server{cfg: cfg, store: store, sessions: NewSessionStore(), cache: NewResponseCache()}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/agent/ping", s.handleAgentPing)
	mux.HandleFunc("/api/agent/report", s.handleAgentReport)
	mux.HandleFunc("/api/admin/login", s.handleAdminLogin)
	mux.HandleFunc("/api/admin/logout", s.handleAdminLogout)
	mux.HandleFunc("/api/admin/me", s.handleAdminMe)
	mux.HandleFunc("/api/admin/settings", s.handleAdminSettings)
	mux.HandleFunc("/api/admin/nodes", s.handleAdminNodes)
	mux.HandleFunc("/api/admin/nodes/export", s.handleAdminNodesExport)
	mux.HandleFunc("/api/admin/nodes/import", s.handleAdminNodesImport)
	mux.HandleFunc("/api/admin/install-command", s.handleAdminInstallCommand)
	mux.HandleFunc("/install/agent-linux.sh", s.handleAgentLinuxInstaller)
	mux.HandleFunc("/install/agent-windows.ps1", s.handleAgentWindowsInstaller)
	mux.HandleFunc("/uninstall/agent-linux.sh", s.handleAgentLinuxUninstaller)
	mux.HandleFunc("/uninstall/agent-windows.ps1", s.handleAgentWindowsUninstaller)
	mux.HandleFunc("/download/", s.handleDownload)
	mux.HandleFunc("/admin", s.handleAdminPage)
	mux.HandleFunc("/admin/", s.handleAdminPage)
	mux.HandleFunc("/config.json", s.handleConfig)
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/info", s.handleInfo)
	mux.HandleFunc("/delete", s.handleDelete)
	mux.HandleFunc("/api/nodes", s.handleNodes)
	mux.HandleFunc("/", s.handleStatic)
	s.http = &http.Server{
		Addr:           cfg.Addr,
		Handler:        withCORS(mux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 16 << 10,
	}
	return s, nil
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !validAdminOrigin(r) {
		http.Error(w, "invalid request origin", http.StatusForbidden)
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !constantEqual(req.Username, s.cfg.AdminUser) || !constantEqual(req.Password, s.cfg.AdminPass) {
		time.Sleep(300 * time.Millisecond)
		http.Error(w, "invalid admin credentials", http.StatusUnauthorized)
		return
	}
	token, err := s.sessions.Create()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, adminCookie(r, token, 24*time.Hour))
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("monitor_admin"); err == nil {
		s.sessions.Delete(cookie.Value)
	}
	c := adminCookie(r, "", -time.Hour)
	http.SetCookie(w, c)
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) handleAdminMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]bool{"authenticated": s.adminAuthorized(r)})
}

func (s *Server) handleAdminNodes(w http.ResponseWriter, r *http.Request) {
	if !s.adminAuthorized(r) {
		http.Error(w, "admin login required", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet && !validAdminOrigin(r) {
		http.Error(w, "invalid request origin", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.store.AdminNodes(s.cfg.OfflineWait))
	case http.MethodPost:
		var req struct {
			NodeID string `json:"node_id"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.NodeID = strings.TrimSpace(req.NodeID)
		if !validNodeID(req.NodeID) {
			http.Error(w, "invalid node_id", http.StatusBadRequest)
			return
		}
		if err := s.store.AddPlannedNode(req.NodeID, s.cfg.MaxNodes); err != nil {
			http.Error(w, err.Error(), http.StatusTooManyRequests)
			return
		}
		s.cache.MarkDirty()
		writeJSON(w, map[string]bool{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleAdminNodesExport(w http.ResponseWriter, r *http.Request) {
	if !s.adminAuthorized(r) {
		http.Error(w, "admin login required", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=monitor-nodes.json")
	writeJSON(w, s.store.ExportNodes())
}

func (s *Server) handleAdminNodesImport(w http.ResponseWriter, r *http.Request) {
	if !s.adminAuthorized(r) {
		http.Error(w, "admin login required", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !validAdminOrigin(r) {
		http.Error(w, "invalid request origin", http.StatusForbidden)
		return
	}
	defer r.Body.Close()
	var backup NodeBackup
	if err := json.NewDecoder(io.LimitReader(r.Body, 10<<20)).Decode(&backup); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	imported, err := s.store.ImportNodes(backup, s.cfg.MaxNodes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.cache.MarkDirty()
	writeJSON(w, map[string]int{"imported": imported})
}

func (s *Server) handleAdminInstallCommand(w http.ResponseWriter, r *http.Request) {
	if !s.adminAuthorized(r) {
		http.Error(w, "admin login required", http.StatusUnauthorized)
		return
	}
	nodeID := strings.TrimSpace(r.URL.Query().Get("node_id"))
	platform := strings.TrimSpace(r.URL.Query().Get("platform"))
	if !validNodeID(nodeID) {
		http.Error(w, "invalid node_id", http.StatusBadRequest)
		return
	}
	token, err := newAgentToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.store.SetNodeToken(nodeID, hashToken(token))
	base := s.externalBase(r)
	linux := fmt.Sprintf("curl -fsSL %s/install/agent-linux.sh | sudo sh -s -- --server %s --token %s --node-id %s", base, base, shellQuote(token), shellQuote(nodeID))
	windows := fmt.Sprintf("powershell -ExecutionPolicy Bypass -Command \"iwr %s/install/agent-windows.ps1 -UseBasicParsing | iex; Install-VpsAgent -Server '%s' -Token '%s' -NodeId '%s'\"", base, base, psQuote(token), psQuote(nodeID))
	linuxUninstall := fmt.Sprintf("curl -fsSL %s/uninstall/agent-linux.sh | sudo sh", base)
	windowsUninstall := fmt.Sprintf("powershell -ExecutionPolicy Bypass -Command \"iwr %s/uninstall/agent-windows.ps1 -UseBasicParsing | iex\"", base)
	if platform == "linux" {
		writeJSON(w, map[string]string{"command": linux})
		return
	}
	if platform == "windows" {
		writeJSON(w, map[string]string{"command": windows})
		return
	}
	if platform == "linux-uninstall" {
		writeJSON(w, map[string]string{"command": linuxUninstall})
		return
	}
	if platform == "windows-uninstall" {
		writeJSON(w, map[string]string{"command": windowsUninstall})
		return
	}
	writeJSON(w, map[string]string{"linux": linux, "windows": windows, "linux_uninstall": linuxUninstall, "windows_uninstall": windowsUninstall})
}

func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}

func (s *Server) handleAgentPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if !s.agentAuthorized(r) {
		http.Error(w, "missing agent identity", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]string{"ok": "true"})
}

func (s *Server) handleAgentReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.agentAuthorized(r) {
		http.Error(w, "missing agent identity", http.StatusUnauthorized)
		return
	}
	defer r.Body.Close()
	var metrics agent.Metrics
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	metrics.NodeID = r.Header.Get("X-Node-ID")
	if !validNodeID(metrics.NodeID) {
		http.Error(w, "invalid node_id", http.StatusBadRequest)
		return
	}
	metrics.Timestamp = time.Now().Unix()
	if err := s.store.UpsertReport(metrics, s.cfg.MaxNodes); err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}
	s.cache.MarkDirty()
	writeJSON(w, map[string]string{"ok": "true"})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	base := s.requestBase(r)
	writeJSON(w, map[string]string{
		"socket":      socketURL(base),
		"apiURL":      base,
		"siteName":    s.store.SiteName(),
		"offlineWait": fmt.Sprintf("%.0f", s.cfg.OfflineWait.Seconds()),
	})
}

func (s *Server) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	if !s.adminAuthorized(r) {
		http.Error(w, "admin login required", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.store.Settings)
	case http.MethodPost:
		if !validAdminOrigin(r) {
			http.Error(w, "invalid request origin", http.StatusForbidden)
			return
		}
		var req Settings
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.SiteName = strings.TrimSpace(req.SiteName)
		if req.SiteName == "" || len(req.SiteName) > 64 {
			http.Error(w, "invalid site_name", http.StatusBadRequest)
			return
		}
		s.store.UpdateSettings(req)
		writeJSON(w, map[string]bool{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.store.InfoList())
	case http.MethodPost:
		var req HostInfo
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !s.adminAuthorized(r) {
			http.Error(w, "admin login required", http.StatusUnauthorized)
			return
		}
		if !validAdminOrigin(r) {
			http.Error(w, "invalid request origin", http.StatusForbidden)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if !validNodeID(req.Name) {
			http.Error(w, "invalid node_id", http.StatusBadRequest)
			return
		}
		s.store.UpsertInfo(req)
		writeJSON(w, map[string]string{"ok": "true"})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !s.adminAuthorized(r) {
		http.Error(w, "admin login required", http.StatusUnauthorized)
		return
	}
	if !validAdminOrigin(r) {
		http.Error(w, "invalid request origin", http.StatusForbidden)
		return
	}
	if !validNodeID(req.Name) {
		http.Error(w, "invalid node_id", http.StatusBadRequest)
		return
	}
	s.store.Delete(req.Name)
	s.cache.MarkDirty()
	writeJSON(w, map[string]string{"ok": "true"})
}

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(adminHTML))
}

func (s *Server) handleAgentLinuxInstaller(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	base := s.externalBase(r)
	_, _ = fmt.Fprintf(w, linuxInstallTemplate, base)
}

func (s *Server) handleAgentWindowsInstaller(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	base := s.externalBase(r)
	_, _ = fmt.Fprintf(w, windowsInstallTemplate, base)
}

func (s *Server) handleAgentLinuxUninstaller(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(linuxUninstallTemplate))
}

func (s *Server) handleAgentWindowsUninstaller(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(windowsUninstallTemplate))
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/download/")
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	data, err := agentBinaries.ReadFile("agent_bins/" + name)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(data)
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	s.writeCachedHosts(w)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, rw, err := upgradeWebSocket(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer conn.Close()
	if err := writeWSBytes(rw, s.cachedHostsJSON()); err != nil {
		return
	}
	for {
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, err := readWS(conn)
		if err != nil {
			return
		}
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := writeWSBytes(rw, s.cachedHostsJSON()); err != nil {
			return
		}
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/") {
		s.handleAdminPage(w, r)
		return
	}
	if r.URL.Path == "/install/agent-linux.sh" {
		s.handleAgentLinuxInstaller(w, r)
		return
	}
	if r.URL.Path == "/install/agent-windows.ps1" {
		s.handleAgentWindowsInstaller(w, r)
		return
	}
	if r.URL.Path == "/uninstall/agent-linux.sh" {
		s.handleAgentLinuxUninstaller(w, r)
		return
	}
	if r.URL.Path == "/uninstall/agent-windows.ps1" {
		s.handleAgentWindowsUninstaller(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/download/") {
		s.handleDownload(w, r)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	data, err := staticFiles.ReadFile("web/dist/" + path)
	if err != nil {
		data, err = staticFiles.ReadFile("web/dist/index.html")
		if err != nil {
			http.Error(w, "frontend is not built; run npm install && npm run build in web", http.StatusNotFound)
			return
		}
	}
	setStaticCache(w, path)
	if ext := filepath.Ext(path); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
	}
	if acceptsGzip(r) && shouldGzip(path) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		_, _ = gz.Write(data)
		return
	}
	w.Write(data)
}

func (s *Server) cachedHostsJSON() []byte {
	return s.cache.Get(func() []byte {
		data, err := json.Marshal(s.store.AkileHosts())
		if err != nil {
			return []byte("[]")
		}
		return data
	})
}

func (s *Server) writeCachedHosts(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(s.cachedHostsJSON())
}

func (s *Server) requestBase(r *http.Request) string {
	if s.cfg.PublicURL != "" {
		return strings.TrimRight(s.cfg.PublicURL, "/")
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Host
	return scheme + "://" + host
}

func (s *Server) externalBase(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Host
	if host != "" && !strings.HasPrefix(host, "127.0.0.1") && !strings.HasPrefix(host, "localhost") {
		return scheme + "://" + host
	}
	return s.requestBase(r)
}

func (s *Server) agentAuthorized(r *http.Request) bool {
	nodeID := strings.TrimSpace(r.Header.Get("X-Node-ID"))
	if !validNodeID(nodeID) {
		return false
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return token != "" && s.store.ValidNodeToken(nodeID, hashToken(token))
}

func (s *Server) adminAuthorized(r *http.Request) bool {
	cookie, err := r.Cookie("monitor_admin")
	if err != nil || cookie.Value == "" {
		return false
	}
	return s.sessions.Valid(cookie.Value)
}

func adminCookie(r *http.Request, value string, maxAge time.Duration) *http.Cookie {
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	return &http.Cookie{
		Name:     "monitor_admin",
		Value:    value,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func constantEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: map[string]time.Time{}}
}

func (s *SessionStore) Create() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = time.Now().Add(24 * time.Hour)
	return token, nil
}

func (s *SessionStore) Valid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expires, ok := s.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(expires) {
		delete(s.sessions, token)
		return false
	}
	return true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func socketURL(base string) string {
	base = strings.TrimRight(base, "/")
	base = strings.TrimPrefix(base, "http://")
	if strings.HasPrefix(base, "https://") {
		return "wss://" + strings.TrimPrefix(base, "https://") + "/ws"
	}
	return "ws://" + base + "/ws"
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Node-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type Store struct {
	mu       sync.RWMutex
	path     string
	Reports  map[string]agent.Metrics `json:"reports"`
	Infos    map[string]HostInfo      `json:"infos"`
	Planned  map[string]PlannedNode   `json:"planned"`
	Settings Settings                 `json:"settings"`
}

type Settings struct {
	SiteName string `json:"site_name"`
}

type PlannedNode struct {
	NodeID    string `json:"node_id"`
	CreatedAt int64  `json:"created_at"`
	TokenHash string `json:"token_hash,omitempty"`
}

type AdminNode struct {
	NodeID    string   `json:"node_id"`
	Online    bool     `json:"online"`
	LastSeen  int64    `json:"last_seen"`
	CreatedAt int64    `json:"created_at"`
	Info      HostInfo `json:"info"`
}

type NodeBackup struct {
	Version    int                `json:"version"`
	ExportedAt int64              `json:"exported_at"`
	Nodes      []NodeBackupRecord `json:"nodes"`
}

type NodeBackupRecord struct {
	NodeID    string   `json:"node_id"`
	CreatedAt int64    `json:"created_at"`
	TokenHash string   `json:"token_hash,omitempty"`
	Info      HostInfo `json:"info"`
}

type HostInfo struct {
	Name       string `json:"name"`
	DueTime    int64  `json:"due_time"`
	BuyURL     string `json:"buy_url"`
	Seller     string `json:"seller"`
	Price      string `json:"price"`
	Cycle      string `json:"cycle"`
	Bandwidth  string `json:"bandwidth"`
	Traffic    string `json:"traffic"`
	Show       bool   `json:"show_purchase_info"`
	AuthSecret string `json:"auth_secret,omitempty"`
}

type AkileHost struct {
	Host      AkileHostMeta  `json:"Host"`
	State     AkileHostState `json:"State"`
	TimeStamp int64          `json:"TimeStamp"`
}

type AkileHostMeta struct {
	Name            string `json:"Name"`
	Hostname        string `json:"Hostname"`
	Platform        string `json:"Platform"`
	PlatformVersion string `json:"PlatformVersion"`
	Kernel          string `json:"Kernel"`
	Arch            string `json:"Arch"`
	Virtualization  string `json:"Virtualization"`
	CPU             []int  `json:"CPU"`
	CPUModel        string `json:"CPUModel"`
	PhysicalCores   int    `json:"PhysicalCores"`
	LogicalCores    int    `json:"LogicalCores"`
	MemTotal        uint64 `json:"MemTotal"`
	SwapTotal       uint64 `json:"SwapTotal"`
}

type AkileHostState struct {
	CPU            float64      `json:"CPU"`
	MemUsed        uint64       `json:"MemUsed"`
	SwapUsed       uint64       `json:"SwapUsed"`
	DiskUsed       uint64       `json:"DiskUsed"`
	DiskTotal      uint64       `json:"DiskTotal"`
	Disks          []agent.Disk `json:"Disks"`
	NetInTransfer  uint64       `json:"NetInTransfer"`
	NetOutTransfer uint64       `json:"NetOutTransfer"`
	NetInSpeed     uint64       `json:"NetInSpeed"`
	NetOutSpeed    uint64       `json:"NetOutSpeed"`
	DiskReadSpeed  uint64       `json:"DiskReadSpeed"`
	DiskWriteSpeed uint64       `json:"DiskWriteSpeed"`
	TCP            int          `json:"TCP"`
	UDP            int          `json:"UDP"`
	Processes      int          `json:"Processes"`
	Load1          float64      `json:"Load1"`
	Load5          float64      `json:"Load5"`
	Load15         float64      `json:"Load15"`
	Uptime         uint64       `json:"Uptime"`
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, Reports: map[string]agent.Metrics{}, Infos: map[string]HostInfo{}, Planned: map[string]PlannedNode{}, Settings: Settings{SiteName: "Monitor Party"}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, s); err != nil {
			return nil, err
		}
	}
	if s.Reports == nil {
		s.Reports = map[string]agent.Metrics{}
	}
	if s.Infos == nil {
		s.Infos = map[string]HostInfo{}
	}
	if s.Planned == nil {
		s.Planned = map[string]PlannedNode{}
	}
	if s.Settings.SiteName == "" {
		s.Settings.SiteName = "Monitor Party"
	}
	return s, nil
}

func (s *Store) SiteName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Settings.SiteName == "" {
		return "Monitor Party"
	}
	return s.Settings.SiteName
}

func (s *Store) UpdateSettings(settings Settings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Settings = settings
	s.saveLocked()
}

func (s *Store) UpsertReport(metrics agent.Metrics, maxNodes int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.Reports[metrics.NodeID]; !exists && len(s.Reports) >= maxNodes {
		return fmt.Errorf("max nodes reached")
	}
	s.Reports[metrics.NodeID] = metrics
	if _, ok := s.Planned[metrics.NodeID]; !ok {
		s.Planned[metrics.NodeID] = PlannedNode{NodeID: metrics.NodeID, CreatedAt: time.Now().Unix()}
	}
	return nil
}

func (s *Store) AddPlannedNode(nodeID string, maxNodes int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.Planned[nodeID]; !exists && len(s.Planned) >= maxNodes {
		return fmt.Errorf("max nodes reached")
	}
	s.Planned[nodeID] = PlannedNode{NodeID: nodeID, CreatedAt: time.Now().Unix()}
	s.saveLocked()
	return nil
}

func (s *Store) SetNodeToken(nodeID, tokenHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	planned := s.Planned[nodeID]
	planned.NodeID = nodeID
	if planned.CreatedAt == 0 {
		planned.CreatedAt = time.Now().Unix()
	}
	planned.TokenHash = tokenHash
	s.Planned[nodeID] = planned
	s.saveLocked()
}

func (s *Store) ValidNodeToken(nodeID, tokenHash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	planned, ok := s.Planned[nodeID]
	if !ok || planned.TokenHash == "" || tokenHash == "" {
		return false
	}
	return constantEqual(planned.TokenHash, tokenHash)
}

func (s *Store) UpsertInfo(info HostInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info.AuthSecret = ""
	s.Infos[info.Name] = info
	s.saveLocked()
}

func (s *Store) Delete(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Reports, name)
	delete(s.Planned, name)
	delete(s.Infos, name)
	s.saveLocked()
}

func (s *Store) InfoList() []HostInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HostInfo, 0, len(s.Infos))
	for _, info := range s.Infos {
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Store) AkileHosts() []AkileHost {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AkileHost, 0, len(s.Planned)+len(s.Reports))
	for _, m := range s.Reports {
		out = append(out, toAkileHost(m))
	}
	for name := range s.Planned {
		if _, ok := s.Reports[name]; ok {
			continue
		}
		out = append(out, offlineAkileHost(name))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Host.Name < out[j].Host.Name })
	return out
}

func (s *Store) AdminNodes(offlineWait time.Duration) []AdminNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().Unix()
	threshold := int64(offlineWait.Seconds())
	seen := map[string]bool{}
	out := make([]AdminNode, 0, len(s.Planned)+len(s.Reports))
	for name, planned := range s.Planned {
		report, hasReport := s.Reports[name]
		lastSeen := int64(0)
		online := false
		if hasReport {
			lastSeen = report.Timestamp
			online = report.Timestamp > 0 && now-report.Timestamp <= threshold
		}
		out = append(out, AdminNode{NodeID: name, Online: online, LastSeen: lastSeen, CreatedAt: planned.CreatedAt, Info: s.Infos[name]})
		seen[name] = true
	}
	for name, report := range s.Reports {
		if seen[name] {
			continue
		}
		online := report.Timestamp > 0 && now-report.Timestamp <= threshold
		out = append(out, AdminNode{NodeID: name, Online: online, LastSeen: report.Timestamp, Info: s.Infos[name]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out
}

func (s *Store) ExportNodes() NodeBackup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := map[string]bool{}
	for name := range s.Planned {
		names[name] = true
	}
	for name := range s.Infos {
		names[name] = true
	}
	for name := range s.Reports {
		names[name] = true
	}
	out := NodeBackup{Version: 1, ExportedAt: time.Now().Unix(), Nodes: make([]NodeBackupRecord, 0, len(names))}
	for name := range names {
		planned := s.Planned[name]
		info := s.Infos[name]
		info.AuthSecret = ""
		out.Nodes = append(out.Nodes, NodeBackupRecord{NodeID: name, CreatedAt: planned.CreatedAt, TokenHash: planned.TokenHash, Info: info})
	}
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].NodeID < out.Nodes[j].NodeID })
	return out
}

func (s *Store) ImportNodes(backup NodeBackup, maxNodes int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if backup.Version == 0 {
		backup.Version = 1
	}
	if backup.Version != 1 {
		return 0, fmt.Errorf("unsupported backup version")
	}
	imported := 0
	now := time.Now().Unix()
	for _, record := range backup.Nodes {
		nodeID := strings.TrimSpace(record.NodeID)
		if nodeID == "" && record.Info.Name != "" {
			nodeID = strings.TrimSpace(record.Info.Name)
		}
		if !validNodeID(nodeID) {
			return imported, fmt.Errorf("invalid node_id: %s", nodeID)
		}
		if _, exists := s.Planned[nodeID]; !exists && len(s.Planned) >= maxNodes {
			return imported, fmt.Errorf("max nodes reached")
		}
		planned := s.Planned[nodeID]
		planned.NodeID = nodeID
		if record.CreatedAt > 0 {
			planned.CreatedAt = record.CreatedAt
		} else if planned.CreatedAt == 0 {
			planned.CreatedAt = now
		}
		if record.TokenHash != "" {
			planned.TokenHash = strings.TrimSpace(record.TokenHash)
		}
		s.Planned[nodeID] = planned
		info := record.Info
		info.Name = nodeID
		info.AuthSecret = ""
		s.Infos[nodeID] = info
		imported++
	}
	if imported > 0 {
		s.saveLocked()
	}
	return imported, nil
}

func (s *Store) saveLocked() {
	if s.path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		log.Printf("save mkdir failed: %v", err)
		return
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Printf("save marshal failed: %v", err)
		return
	}
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		log.Printf("save failed: %v", err)
	}
}

func toAkileHost(m agent.Metrics) AkileHost {
	diskUsed := uint64(0)
	diskTotal := uint64(0)
	for _, disk := range m.Disks {
		diskUsed += disk.Used
		diskTotal += disk.Total
	}
	platform := m.OS
	if m.OSName != "" {
		platform = m.OSName
	}
	if platform == "" {
		platform = "unknown"
	}
	conns := agent.Connections{}
	if m.Conns != nil {
		conns = *m.Conns
	}
	return AkileHost{
		Host: AkileHostMeta{
			Name:            m.NodeID,
			Hostname:        m.Hostname,
			Platform:        platform,
			PlatformVersion: m.Kernel,
			Kernel:          m.Kernel,
			Arch:            m.Arch,
			Virtualization:  m.Virtualization,
			CPU:             make([]int, m.CPU.Cores),
			CPUModel:        m.CPU.ModelName,
			PhysicalCores:   m.CPU.PhysicalCores,
			LogicalCores:    m.CPU.Cores,
			MemTotal:        m.Memory.Total,
			SwapTotal:       m.Swap.Total,
		},
		State: AkileHostState{
			CPU:            m.CPU.UsagePercent,
			MemUsed:        m.Memory.Used,
			SwapUsed:       m.Swap.Used,
			DiskUsed:       diskUsed,
			DiskTotal:      diskTotal,
			Disks:          m.Disks,
			NetInTransfer:  m.Network.RxBytes,
			NetOutTransfer: m.Network.TxBytes,
			NetInSpeed:     m.Network.RxRate,
			NetOutSpeed:    m.Network.TxRate,
			DiskReadSpeed:  m.DiskIO.ReadRate,
			DiskWriteSpeed: m.DiskIO.WriteRate,
			TCP:            conns.TCP,
			UDP:            conns.UDP,
			Processes:      m.Processes,
			Load1:          m.Load.Load1,
			Load5:          m.Load.Load5,
			Load15:         m.Load.Load15,
			Uptime:         m.Uptime,
		},
		TimeStamp: m.Timestamp,
	}
}

func offlineAkileHost(name string) AkileHost {
	return AkileHost{
		Host:      AkileHostMeta{Name: name, Platform: "pending", PlatformVersion: "", CPU: []int{}, MemTotal: 1},
		State:     AkileHostState{},
		TimeStamp: 0,
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func psQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func validNodeID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > 96 {
		return false
	}
	return !strings.ContainsAny(value, "\x00\r\n\t/\\'\"`$;&|<>!*?[]{}()")
}

func isWeakSecret(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || value == "change-me"
}

func newAgentToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf[:]), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func cleanPublicURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", errors.New("PUBLIC_URL must be an absolute http or https URL")
	}
	if u.Scheme != "https" && !strings.HasPrefix(u.Host, "127.0.0.1") && !strings.HasPrefix(u.Host, "localhost") {
		return "", errors.New("PUBLIC_URL must use https outside localhost")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func validAdminOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, *bufioWriter, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, nil, errors.New("not websocket")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, nil, errors.New("missing websocket key")
	}
	h, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijacking unsupported")
	}
	conn, rw, err := h.Hijack()
	if err != nil {
		return nil, nil, err
	}
	accept := websocketAccept(key)
	_, err = fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", accept)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return nil, nil, err
	}
	return conn, &bufioWriter{conn: conn}, nil
}

type bufioWriter struct{ conn net.Conn }

func (w *bufioWriter) Write(p []byte) (int, error) { return w.conn.Write(p) }

func websocketAccept(key string) string {
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h[:])
}

func readWS(conn net.Conn) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	opcode := header[0] & 0x0f
	if opcode == 8 {
		return nil, io.EOF
	}
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7f)
	if length == 126 {
		buf := make([]byte, 2)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return nil, err
		}
		length = uint64(binary.BigEndian.Uint16(buf))
	} else if length == 127 {
		buf := make([]byte, 8)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return nil, err
		}
		length = binary.BigEndian.Uint64(buf)
	}
	if length > 1<<20 {
		return nil, errors.New("websocket frame too large")
	}
	mask := make([]byte, 4)
	if masked {
		if _, err := io.ReadFull(conn, mask); err != nil {
			return nil, err
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return payload, nil
}

func writeWS(w io.Writer, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	header := []byte{0x81}
	if len(payload) < 126 {
		header = append(header, byte(len(payload)))
	} else if len(payload) <= 65535 {
		header = append(header, 126, byte(len(payload)>>8), byte(len(payload)))
	} else {
		header = append(header, 127)
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(len(payload)))
		header = append(header, buf[:]...)
	}
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}

func writeWSBytes(w io.Writer, payload []byte) error {
	header := []byte{0x81}
	if len(payload) < 126 {
		header = append(header, byte(len(payload)))
	} else if len(payload) <= 65535 {
		header = append(header, 126, byte(len(payload)>>8), byte(len(payload)))
	} else {
		header = append(header, 127)
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(len(payload)))
		header = append(header, buf[:]...)
	}
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

type ResponseCache struct {
	mu      sync.Mutex
	dirty   bool
	expires time.Time
	data    []byte
}

func NewResponseCache() *ResponseCache {
	return &ResponseCache{dirty: true}
}

func (c *ResponseCache) MarkDirty() {
	c.mu.Lock()
	c.dirty = true
	c.mu.Unlock()
}

func (c *ResponseCache) Get(build func() []byte) []byte {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dirty && now.Before(c.expires) && c.data != nil {
		return c.data
	}
	c.data = build()
	c.expires = now.Add(time.Second)
	c.dirty = false
	return c.data
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func shouldGzip(path string) bool {
	switch filepath.Ext(path) {
	case ".html", ".css", ".js", ".json", ".svg":
		return true
	default:
		return false
	}
}

func setStaticCache(w http.ResponseWriter, path string) {
	if path == "index.html" || path == "config.json" {
		w.Header().Set("Cache-Control", "no-cache")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
}

func hasStaticBuild() bool {
	_, err := fs.Stat(staticFiles, "web/dist/index.html")
	return err == nil
}
