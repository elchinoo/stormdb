package e2e

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestTPCCEndToEnd(t *testing.T) {
	// Build core and plugins from project root
	buildCmd := exec.Command("make", "build-all")
	buildCmd.Dir = ".."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build-all failed: %v\n%s", err, output)
	}

	// Start PostgreSQL via Docker Compose (service "postgres")
	upCmd := exec.Command("docker-compose", "up", "-d", "postgres")
	upCmd.Dir = ".."
	if output, err := upCmd.CombinedOutput(); err != nil {
		t.Skipf("docker-compose up failed, skipping e2e test: %v\n%s", err, output)
	}
	// Ensure docker-compose is torn down after test
	defer func() {
		downCmd := exec.Command("docker-compose", "down")
		downCmd.Dir = ".."
		_ = downCmd.Run()
	}()

	// Wait for PostgreSQL readiness on localhost:5432
	waitTimeout := time.Now().Add(30 * time.Second)
	var conn net.Conn
	var err error
	for time.Now().Before(waitTimeout) {
		conn, err = net.Dial("tcp", "localhost:5432")
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(1 * time.Second)
	}
	if conn == nil && err != nil {
		t.Fatal("timeout waiting for PostgreSQL readiness")
	}

	// Start StormDB with plugins
	// Start StormDB with plugins from project root
	cmd := exec.Command("./build/stormdb")
	cmd.Dir = ".."
	cmd.Env = append(os.Environ(), "STORMDB_PLUGIN_DIR=./build/plugins")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting stormdb failed: %v", err)
	}
	defer func() {
		// Send interrupt and wait
		_ = cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	}()

	// Wait for health endpoint
	client := &http.Client{Timeout: 1 * time.Second}
	healthURL := "http://localhost:8080/health"
	healthTimeout := time.Now().Add(30 * time.Second)
	for time.Now().Before(healthTimeout) {
		resp, err := client.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}
		time.Sleep(1 * time.Second)
	}

	// Create TPC-C test run
	body := map[string]interface{}{
		"plugin_name": "tpcc-scalability",
		"name":        "CI TPC-C Test",
		"description": "End-to-end test",
		"config": map[string]interface{}{
			"host":        "localhost",
			"port":        5432,
			"database":    "stormdb",
			"username":    "postgres",
			"password":    "postgres",
			"ssl_mode":    "disable",
			"scale":       1,
			"connections": []int{1},
			"duration":    "5s",
			"warmup_time": "1s",
		},
	}
	data, _ := json.Marshal(body)
	resp, err := client.Post("http://localhost:8080/test-runs", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("creating test-run failed: %v", err)
	}
	defer resp.Body.Close()
	respData, _ := ioutil.ReadAll(resp.Body)
	var respJSON map[string]interface{}
	if err := json.Unmarshal(respData, &respJSON); err != nil {
		t.Fatalf("invalid create response: %s", respData)
	}
	id, ok := respJSON["id"].(string)
	if !ok {
		t.Fatalf("test id not found in response: %s", respData)
	}

	// Poll test-run status
	statusURL := "http://localhost:8080/test-runs/" + id
	statusTimeout := time.Now().Add(60 * time.Second)
	for time.Now().Before(statusTimeout) {
		resp, err := client.Get(statusURL)
		if err != nil {
			t.Fatalf("status request failed: %v", err)
		}
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		var statusJSON map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &statusJSON); err != nil {
			t.Fatalf("invalid status response: %s", bodyBytes)
		}
		if status, _ := statusJSON["status"].(string); status != "running" {
			if status == "completed" || status == "success" {
				return
			}
			t.Fatalf("test-run ended with status: %v", status)
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("timed out waiting for test-run to complete")
}
