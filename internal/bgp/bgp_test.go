package bgp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{}

func TestMultipleCollectors(t *testing.T) {
	// Create a wait group to track successful subscriptions
	var wg sync.WaitGroup
	var mu sync.Mutex
	subscribedCollectors := make(map[string]bool)

	// Expected collectors
	collectors := []string{"rrc00", "rrc01", "rrc02"}
	for range collectors {
		wg.Add(1)
	}

	// Create a mock WebSocket server that handles multiple connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				break
			}

			// Check for subscribe message
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err == nil {
				if msg["type"] == "ris_subscribe" {
					if data, ok := msg["data"].(map[string]interface{}); ok {
						if host, ok := data["host"].(string); ok {
							mu.Lock()
							if !subscribedCollectors[host] {
								subscribedCollectors[host] = true
								wg.Done()
							}
							mu.Unlock()
						}
					}
				}
			}
		}
	}))
	defer server.Close()

	// Replace http:// with ws:// for the WebSocket connection
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Initialize Monitor with multiple mock collectors
	monitor := New(wsURL, collectors, []int{29049}, nil, logger)

	// Run the monitor with a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitor.Run(ctx)

	// Wait for all collectors to subscribe, or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success! All collectors subscribed concurrently.
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for all collectors to subscribe")
	}

	mu.Lock()
	defer mu.Unlock()
	for _, coll := range collectors {
		if !subscribedCollectors[coll] {
			t.Errorf("collector %s did not subscribe", coll)
		}
	}
}
