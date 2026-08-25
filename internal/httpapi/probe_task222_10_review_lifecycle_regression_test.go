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

func TestReviewKeepsExplicitReviewingLifecycleState(t *testing.T) {
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
	post := func(path string, body any) *http.Response {
		data, _ := json.Marshal(body)
		resp, err := ts.Client().Post(ts.URL+path, "application/json", bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	resp := post("/api/trials", map[string]string{"code": "REVIEW-LIFECYCLE", "cable_name": "cable", "voltage_class": "110kV"})
	var trial struct{ ID string `json:"id"` }
	if err := json.NewDecoder(resp.Body).Decode(&trial); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	for _, action := range []string{"start", "finish"} {
		resp = post("/api/trials/"+trial.ID+"/"+action, nil)
		if resp.StatusCode >= 300 {
			t.Fatalf("%s status=%d", action, resp.StatusCode)
		}
		resp.Body.Close()
	}
	resp = post("/api/trials/"+trial.ID+"/review", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("review status=%d, want 200", resp.StatusCode)
	}
	var reviewed struct{ Status string `json:"status"` }
	if err := json.NewDecoder(resp.Body).Decode(&reviewed); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if reviewed.Status != "reviewing" {
		t.Fatalf("review state=%s, want reviewing", reviewed.Status)
	}
}
