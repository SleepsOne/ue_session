package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"sessionmgr/internal/domain"
)

const baseURL = "http://localhost:8080/api/v1"

func main() {
	fmt.Println("=== UE Session Manager Demo ===")
	fmt.Println()

	// Test health endpoint
	fmt.Println("1. Testing health endpoint...")
	if err := testHealth(); err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}
	fmt.Println("✓ Health check passed")
	fmt.Println()

	// Test session creation
	fmt.Println("2. Testing session creation...")
	session := &domain.Session{
		TMSI:         "12345678",
		IMSI:         "123456789012345",
		MSISDN:       "1234567890",
		GNBID:        "gNB001",
		TAI:          "TAI001",
		UEState:      "REGISTERED",
		Capabilities: []string{"5G", "4G"},
		SecurityCtx: domain.SecurityContext{
			KAMF:                 "test-kamf-123",
			Algorithm:            "AES",
			KeySetID:             "1",
			NextHopChainingCount: 1,
		},
	}

	if err := createSession(session); err != nil {
		log.Printf("Session creation failed: %v", err)
		return
	}
	fmt.Println("✓ Session created successfully")
	fmt.Println()

	// Test session retrieval
	fmt.Println("3. Testing session retrieval...")
	retrievedSession, err := getSession(session.TMSI)
	if err != nil {
		log.Printf("Session retrieval failed: %v", err)
		return
	}
	fmt.Printf("✓ Session retrieved: TMSI=%s, IMSI=%s, MSISDN=%s\n",
		retrievedSession.TMSI, retrievedSession.IMSI, retrievedSession.MSISDN)
	fmt.Println()

	// Test session update
	fmt.Println("4. Testing session update...")
	session.GNBID = "gNB002"
	session.TAI = "TAI002"
	if err := updateSession(session); err != nil {
		log.Printf("Session update failed: %v", err)
		return
	}
	fmt.Println("✓ Session updated successfully")
	fmt.Println()

	// Test session query by IMSI
	fmt.Println("5. Testing session query by IMSI...")
	sessions, err := querySessionsByIMSI(session.IMSI)
	if err != nil {
		log.Printf("Session query failed: %v", err)
		return
	}
	fmt.Printf("✓ Found %d sessions for IMSI %s\n", len(sessions), session.IMSI)
	fmt.Println()

	// Test session query by MSISDN
	fmt.Println("6. Testing session query by MSISDN...")
	sessions, err = querySessionsByMSISDN(session.MSISDN)
	if err != nil {
		log.Printf("Session query failed: %v", err)
		return
	}
	fmt.Printf("✓ Found %d sessions for MSISDN %s\n", len(sessions), session.MSISDN)
	fmt.Println()

	// Test session TTL renewal
	fmt.Println("7. Testing session TTL renewal...")
	if err := renewSession(session.TMSI); err != nil {
		log.Printf("Session renewal failed: %v", err)
		return
	}
	fmt.Println("✓ Session TTL renewed successfully")
	fmt.Println()

	// Test session deletion
	fmt.Println("8. Testing session deletion...")
	if err := deleteSession(session.TMSI); err != nil {
		log.Printf("Session deletion failed: %v", err)
		return
	}
	fmt.Println("✓ Session deleted successfully")
	fmt.Println()

	// Test getting deleted session
	fmt.Println("9. Testing retrieval of deleted session...")
	_, err = getSession(session.TMSI)
	if err == nil {
		log.Printf("Expected error when getting deleted session")
		return
	}
	fmt.Println("✓ Correctly received error for deleted session")
	fmt.Println()

	fmt.Println("=== Demo completed successfully! ===")
}

func testHealth() error {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

func createSession(session *domain.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	resp, err := http.Post(baseURL+"/sessions", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("session creation returned status %d", resp.StatusCode)
	}

	return nil
}

func getSession(tmsi string) (*domain.Session, error) {
	resp, err := http.Get(baseURL + "/sessions/" + tmsi)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("session retrieval returned status %d", resp.StatusCode)
	}

	var response struct {
		Session *domain.Session `json:"session"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Session, nil
}

func updateSession(session *domain.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", baseURL+"/sessions/"+session.TMSI, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session update returned status %d", resp.StatusCode)
	}

	return nil
}

func querySessionsByIMSI(imsi string) ([]*domain.Session, error) {
	resp, err := http.Get(baseURL + "/sessions?imsi=" + imsi)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("session query returned status %d", resp.StatusCode)
	}

	var response struct {
		Sessions []*domain.Session `json:"sessions"`
		Count    int               `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Sessions, nil
}

func querySessionsByMSISDN(msisdn string) ([]*domain.Session, error) {
	resp, err := http.Get(baseURL + "/sessions?msisdn=" + msisdn)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("session query returned status %d", resp.StatusCode)
	}

	var response struct {
		Sessions []*domain.Session `json:"sessions"`
		Count    int               `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Sessions, nil
}

func renewSession(tmsi string) error {
	req, err := http.NewRequest("POST", baseURL+"/sessions/"+tmsi+"/renew", nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session renewal returned status %d", resp.StatusCode)
	}

	return nil
}

func deleteSession(tmsi string) error {
	req, err := http.NewRequest("DELETE", baseURL+"/sessions/"+tmsi, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session deletion returned status %d", resp.StatusCode)
	}

	return nil
}
