package main

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Device struct {
	ID       string    `json:"id"`
	Model    string    `json:"model"`
	Android  string    `json:"android"`
	App      string    `json:"app"`
	Rooted   bool      `json:"rooted"`
	LastSeen time.Time `json:"lastSeen"`
	conn     *websocket.Conn
}
type Hub struct {
	sync.RWMutex
	devices  map[string]*Device
	token    string
	sessions map[string]time.Time
}
type message struct {
	Type     string `json:"type"`
	Token    string `json:"token,omitempty"`
	DeviceID string `json:"deviceId,omitempty"`
	Model    string `json:"model,omitempty"`
	Android  string `json:"android,omitempty"`
	App      string `json:"app,omitempty"`
	Rooted   bool   `json:"rooted,omitempty"`
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var page = template.Must(template.New("dashboard").Parse(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>KMoni</title><style>body{font:16px system-ui;margin:2rem;background:#f6f7f9;color:#18202a}main{max-width:900px;margin:auto}section{background:white;border-radius:10px;padding:1rem;margin:1rem 0;box-shadow:0 1px 4px #0001}code{background:#eef;padding:.15rem .3rem;border-radius:4px}.muted{color:#667}button{margin:.5rem 0;padding:.4rem .7rem}</style></head><body><main><h1>KMoni</h1><p class="muted">Connected personal devices</p><form method="post" action="/logout"><button>Log out</button></form><section id="devices">Loading…</section><script>async function load(){const r=await fetch('/api/devices');if(r.status===401){document.getElementById('devices').innerHTML='<form method="post" action="/login"><input name="token" type="password" placeholder="Pairing token" required><button>Sign in</button></form>';return}const d=await r.json();document.getElementById('devices').innerHTML=d.length?d.map(function(x){return '<article><h2>'+x.id+'</h2><div>'+x.model+' · Android '+x.android+' · app '+x.app+'</div><div>Root: '+(x.rooted?'yes':'no')+' · last seen '+new Date(x.lastSeen).toLocaleString()+'</div></article>'}).join(''):'<p>No devices connected.</p>'}load();setInterval(load,5000)</script></section></main></body></html>`))

func main() {
	loadConfig(env("KMONI_CONFIG", "config.env"))
	addr := flag.String("addr", env("KMONI_ADDR", ":8080"), "listen address")
	flag.Parse()
	token := os.Getenv("KMONI_PAIRING_TOKEN")
	if len(token) < 16 {
		log.Fatal("KMONI_PAIRING_TOKEN must be at least 16 characters")
	}
	h := &Hub{devices: map[string]*Device{}, sessions: map[string]time.Time{}, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.dashboard)
	mux.HandleFunc("/login", h.login)
	mux.HandleFunc("/logout", h.logout)
	mux.HandleFunc("/api/devices", h.apiDevices)
	mux.HandleFunc("/ws", h.websocket)
	s := &http.Server{Addr: *addr, Handler: securityHeaders(mux), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("KMoni listening on %s", *addr)
	log.Fatal(s.ListenAndServe())
}

func loadConfig(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && os.Getenv(strings.TrimSpace(parts[0])) == "" {
			os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'unsafe-inline'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
func (h *Hub) dashboard(w http.ResponseWriter, r *http.Request) { page.Execute(w, nil) }
func (h *Hub) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	if !same(r.FormValue("token"), h.token) {
		http.Error(w, "invalid token", 401)
		return
	}
	b := make([]byte, 32)
	rand.Read(b)
	id := hex.EncodeToString(b)
	h.Lock()
	h.sessions[id] = time.Now().Add(24 * time.Hour)
	h.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "kmoni_session", Value: id, Path: "/", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: 86400})
	http.Redirect(w, r, "/", 303)
}

func (h *Hub) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("kmoni_session"); err == nil {
		h.Lock()
		delete(h.sessions, c.Value)
		h.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "kmoni_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func (h *Hub) authorized(r *http.Request) bool {
	c, e := r.Cookie("kmoni_session")
	if e != nil {
		return false
	}
	h.RLock()
	until, ok := h.sessions[c.Value]
	h.RUnlock()
	return ok && time.Now().Before(until)
}
func (h *Hub) apiDevices(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	h.RLock()
	out := make([]Device, 0, len(h.devices))
	for _, d := range h.devices {
		x := *d
		x.conn = nil
		out = append(out, x)
	}
	h.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
func same(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func (h *Hub) websocket(w http.ResponseWriter, r *http.Request) {
	c, e := upgrader.Upgrade(w, r, nil)
	if e != nil {
		return
	}
	defer c.Close()
	c.SetReadDeadline(time.Now().Add(15 * time.Second))
	var m message
	if e = c.ReadJSON(&m); e != nil || m.Type != "pair" || !same(m.Token, h.token) || m.DeviceID == "" {
		c.WriteJSON(map[string]string{"error": "pairing failed"})
		return
	}
	d := &Device{ID: m.DeviceID, Model: m.Model, Android: m.Android, App: m.App, Rooted: m.Rooted, LastSeen: time.Now(), conn: c}
	h.Lock()
	if old := h.devices[d.ID]; old != nil && old.conn != nil {
		old.conn.Close()
	}
	h.devices[d.ID] = d
	h.Unlock()
	c.WriteJSON(map[string]string{"type": "paired"})
	c.SetReadDeadline(time.Time{})
	for {
		var x message
		if e = c.ReadJSON(&x); e != nil {
			break
		}
		if x.Type == "status" {
			h.Lock()
			d.LastSeen = time.Now()
			d.Rooted = x.Rooted
			h.Unlock()
		}
	}
	h.Lock()
	if h.devices[d.ID] == d {
		delete(h.devices, d.ID)
	}
	h.Unlock()
}
