package handlers

import "testing"

func TestDecodeFeatures(t *testing.T) {
	features := decodeFeatures([]byte(`{"feature.local_switch":true,"feature.api_relay":false}`))
	if !features["feature.local_switch"] {
		t.Fatal("expected local switching to be enabled")
	}
	if features["feature.api_relay"] {
		t.Fatal("expected relay to be disabled")
	}
}

func TestFeatureForPlatform(t *testing.T) {
	tests := map[string]string{
		"codex":          "platform.codex",
		"claude":         "platform.claude_manager",
		"github-copilot": "platform.github-copilot",
	}
	for input, expected := range tests {
		if actual := featureForPlatform(input); actual != expected {
			t.Fatalf("featureForPlatform(%q) = %q, want %q", input, actual, expected)
		}
	}
}
