package llm

import "testing"

func TestBaseURLOrDefault_Empty_ReturnsDefault(t *testing.T) {
	if got := BaseURLOrDefault(""); got != DefaultBaseURL {
		t.Fatalf("BaseURLOrDefault(%q) = %q, want %q", "", got, DefaultBaseURL)
	}
}

func TestBaseURLOrDefault_Set_ReturnsConfigured(t *testing.T) {
	const url = "https://example.test/v1"
	if got := BaseURLOrDefault(url); got != url {
		t.Fatalf("BaseURLOrDefault(%q) = %q, want %q", url, got, url)
	}
}
