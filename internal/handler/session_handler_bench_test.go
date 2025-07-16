package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

// Session payload struct matching your domain.Session
// Only required fields for creation are included
// Add more fields if your API requires them

type SecurityContext struct {
	KAMF                 string `json:"kamf"`
	Algorithm            string `json:"algorithm"`
	KeySetID             string `json:"keyset_id"`
	NextHopChainingCount int    `json:"next_hop_chaining_count"`
}

type Session struct {
	TMSI         string          `json:"tmsi"`
	IMSI         string          `json:"imsi"`
	MSISDN       string          `json:"msisdn"`
	GNBID        string          `json:"gnb_id"`
	TAI          string          `json:"tai"`
	UEState      string          `json:"ue_state"`
	Capabilities []string        `json:"capabilities"`
	SecurityCtx  SecurityContext `json:"security_context"`
}

// Set the server address here (or override with SERVER_ADDR env var)
var serverAddr = "http://localhost:8080"

func init() {
	if addr := os.Getenv("SERVER_ADDR"); addr != "" {
		serverAddr = addr
	}
}

func randomSession(i int) Session {
	return Session{
		TMSI:         fmt.Sprintf("TMSI%08d", i),
		IMSI:         fmt.Sprintf("IMSI%015d", i),
		MSISDN:       fmt.Sprintf("MSISDN%010d", i),
		GNBID:        fmt.Sprintf("gNB%03d", i%100),
		TAI:          fmt.Sprintf("TAI%03d", i%100),
		UEState:      "REGISTERED",
		Capabilities: []string{"5G", "4G"},
		SecurityCtx: SecurityContext{
			KAMF:                 "test-kamf",
			Algorithm:            "AES",
			KeySetID:             strconv.Itoa(i % 10),
			NextHopChainingCount: rand.Intn(5),
		},
	}
}

func BenchmarkSessionHandler_Create(b *testing.B) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := serverAddr + "/api/v1/sessions"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(1000000)
		for pb.Next() {
			sess := randomSession(i)
			payload, err := json.Marshal(sess)
			if err != nil {
				b.Errorf("failed to marshal session: %v", err)
				continue
			}
			req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
			if err != nil {
				b.Errorf("failed to create request: %v", err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				b.Errorf("request failed: %v", err)
				continue
			}
			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
				b.Errorf("unexpected status: %d", resp.StatusCode)
			}
			resp.Body.Close()
			i++
		}
	})
}
