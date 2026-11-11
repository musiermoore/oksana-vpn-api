package vless

import (
	"encoding/json"
	"testing"
)

func TestBuildLinksReturnsOnlyRequestedClientLinks(t *testing.T) {
	response := BuildLinks("abc-123", "https://connect.example/")
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 4 {
		t.Fatalf("response has %d keys; want exactly four: %s", len(payload), encoded)
	}

	want := map[string]string{
		"happ_deep_link":     "https://connect.example/deep-link/happ?token=abc-123",
		"incy_deep_link":     "https://connect.example/deep-link/incy?token=abc-123",
		"v2raytun_deep_link": "https://connect.example/deep-link/v2raytun?token=abc-123",
	}
	for key, expected := range want {
		var actual string
		if err := json.Unmarshal(payload[key], &actual); err != nil {
			t.Fatalf("decode %s: %v", key, err)
		}
		if actual != expected {
			t.Errorf("%s=%q; want %q", key, actual, expected)
		}
	}
	var showRawLink bool
	if err := json.Unmarshal(payload["show_raw_link"], &showRawLink); err != nil {
		t.Fatalf("decode show_raw_link: %v", err)
	}
	if showRawLink {
		t.Fatal("show_raw_link must always be false")
	}
}
