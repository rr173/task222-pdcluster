package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/service"
	"task222-pdcluster/internal/store"
)

func TestCoreAPIRoundTripPersists(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pdcluster.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	app, err := service.New(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}

	ts := httptest.NewServer(New(app).Handler())
	client := ts.Client()

	status, body := doJSON(t, client, http.MethodGet, ts.URL+"/api/health", nil)
	if status != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("health response: status=%d body=%v", status, body)
	}

	status, body = doJSON(t, client, http.MethodPost, ts.URL+"/api/trials", map[string]any{
		"code": "HTTP-E2E-001", "cable_name": "110kV cable", "voltage_class": "110kV",
	})
	if status != http.StatusCreated {
		t.Fatalf("create trial status=%d body=%v", status, body)
	}
	var trial model.Trial
	decodeMap(t, body, &trial)
	if trial.ID == "" || trial.Status != model.TrialPreparing {
		t.Fatalf("created trial=%+v", trial)
	}

	status, body = doJSON(t, client, http.MethodPost, ts.URL+fmt.Sprintf("/api/trials/%s/reference", trial.ID), map[string]any{
		"freq_hz": 50, "zero_time_ns": 0,
	})
	if status != http.StatusCreated || body["trial_id"] != trial.ID {
		t.Fatalf("reference response: status=%d body=%v", status, body)
	}

	status, body = doJSON(t, client, http.MethodPost, ts.URL+fmt.Sprintf("/api/trials/%s/channels", trial.ID), map[string]any{
		"name": "CH0", "index": 0,
	})
	if status != http.StatusCreated {
		t.Fatalf("channel response: status=%d body=%v", status, body)
	}
	var channel model.Channel
	decodeMap(t, body, &channel)

	status, _ = doJSON(t, client, http.MethodPost, ts.URL+fmt.Sprintf("/api/trials/%s/start", trial.ID), nil)
	if status != http.StatusOK {
		t.Fatalf("start status=%d", status)
	}
	status, body = doJSON(t, client, http.MethodPost, ts.URL+fmt.Sprintf("/api/trials/%s/pulses", trial.ID), map[string]any{
		"channel_id": channel.ID, "channel_index": 0, "seq": 7, "time_ns": 1000000, "amplitude_mv": 25,
	})
	if status != http.StatusCreated || body["inserted"] != float64(1) {
		t.Fatalf("pulse response: status=%d body=%v", status, body)
	}
	resp, err := client.Get(ts.URL + fmt.Sprintf("/api/trials/%s/pulses", trial.ID))
	if err != nil {
		t.Fatal(err)
	}
	var pulses []model.Pulse
	if err := json.NewDecoder(resp.Body).Decode(&pulses); err != nil {
		resp.Body.Close()
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || len(pulses) != 1 {
		t.Fatalf("pulse list status=%d pulses=%v", resp.StatusCode, pulses)
	}

	ts.Close()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db2, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	app2, err := service.New(db2)
	if err != nil {
		t.Fatal(err)
	}
	ts2 := httptest.NewServer(New(app2).Handler())
	defer ts2.Close()
	status, body = doJSON(t, ts2.Client(), http.MethodGet, ts2.URL+fmt.Sprintf("/api/trials/%s", trial.ID), nil)
	if status != http.StatusOK {
		t.Fatalf("reopened trial status=%d body=%v", status, body)
	}
	var restored model.Trial
	decodeMap(t, body, &restored)
	if restored.ID != trial.ID || restored.Status != model.TrialAcquiring {
		t.Fatalf("restored trial=%+v", restored)
	}
}

func doJSON(t *testing.T, client *http.Client, method, url string, requestBody any) (int, map[string]any) {
	t.Helper()
	var body io.Reader
	if requestBody != nil {
		data, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, decoded
}

func decodeMap(t *testing.T, body map[string]any, dst any) {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatal(err)
	}
}
