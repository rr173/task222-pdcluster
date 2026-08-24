package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task222-pdcluster/internal/service"
	"task222-pdcluster/internal/store"
)

func TestPulseBatchIsAtomicWhenOneEventIsInvalid(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(app).Handler())
	defer ts.Close()
	client := ts.Client()

	post := func(path string, body any) *http.Response {
		data, _ := json.Marshal(body)
		resp, err := client.Post(ts.URL+path, "application/json", bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	resp := post("/api/trials", map[string]string{"code": "BATCH-ATOMIC", "cable_name": "cable", "voltage_class": "110kV"})
	var trial struct{ ID string `json:"id"` }
	if err := json.NewDecoder(resp.Body).Decode(&trial); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	resp = post("/api/trials/"+trial.ID+"/channels", map[string]any{"name": "ch0", "index": 0})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("channel status=%d", resp.StatusCode)
	}
	var ch struct{ ID string `json:"id"` }
	if err := json.NewDecoder(resp.Body).Decode(&ch); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp = post("/api/trials/"+trial.ID+"/pulses/batch", map[string]any{"pulses": []any{
		map[string]any{"channel_id": ch.ID, "channel_index": 0, "seq": 1, "time_ns": 1000, "amplitude_mv": 12.0},
		map[string]any{"channel_id": ch.ID, "channel_index": 0, "seq": 2, "time_ns": 2000, "amplitude_mv": -1.0},
	}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("batch status=%d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = client.Get(ts.URL + "/api/trials/" + trial.ID + "/pulses")
	if err != nil {
		t.Fatal(err)
	}
	var pulses []any
	if err := json.NewDecoder(resp.Body).Decode(&pulses); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(pulses) != 0 {
		t.Fatalf("invalid batch partially persisted %d pulse(s)", len(pulses))
	}
}
