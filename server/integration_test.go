package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSimulatedDevice(t *testing.T) {
	h := &Hub{devices: map[string]*Device{}, sessions: map[string]time.Time{}, token: "local-test-token-123456"}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.websocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()
	c, _, err := websocket.DefaultDialer.Dial("ws"+ts.URL[4:]+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.WriteJSON(map[string]any{"type": "pair", "token": h.token, "deviceId": "sim-1", "model": "Simulator", "android": "35", "app": "test", "rooted": true}); err != nil {
		t.Fatal(err)
	}
	var response map[string]string
	if err := c.ReadJSON(&response); err != nil {
		t.Fatal(err)
	}
	if response["type"] != "paired" {
		t.Fatalf("pair response: %#v", response)
	}
	if err := c.WriteJSON(map[string]any{"type": "status", "rooted": true}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	h.RLock()
	d, ok := h.devices["sim-1"]
	h.RUnlock()
	if !ok || d.Model != "Simulator" || !d.Rooted {
		t.Fatalf("device not registered: %#v", d)
	}
}
