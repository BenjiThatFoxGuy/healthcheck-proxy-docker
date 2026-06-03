package main

import (
	"net"
	"os"
	"testing"
	"time"
)

func mustParseDuration(t *testing.T, s string) time.Duration {
	t.Helper()
	d, err := time.ParseDuration(s)
	if err != nil {
		t.Fatalf("invalid duration %q: %v", s, err)
	}
	return d
}

func TestTargetFromEnvDefaults(t *testing.T) {
	os.Unsetenv("TARGET_HOST")
	os.Unsetenv("TARGET_PORT")
	os.Unsetenv("TIMEOUT")

	addr, timeout, err := targetFromEnv()
	if err != nil {
		t.Fatalf("expected no error with defaults, got: %v", err)
	}
	if addr != "localhost:80" {
		t.Errorf("expected 'localhost:80', got %q", addr)
	}
	if timeout != mustParseDuration(t, "2s") {
		t.Errorf("expected 2s timeout, got %v", timeout)
	}
}

func TestTargetFromEnvCustom(t *testing.T) {
	os.Setenv("TARGET_HOST", "redis.example.com")
	os.Setenv("TARGET_PORT", "6379")
	os.Setenv("TIMEOUT", "5s")
	defer func() {
		os.Unsetenv("TARGET_HOST")
		os.Unsetenv("TARGET_PORT")
		os.Unsetenv("TIMEOUT")
	}()

	addr, _, err := targetFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "redis.example.com:6379" {
		t.Errorf("expected 'redis.example.com:6379', got %q", addr)
	}
}

func TestTargetFromEnvInvalidPort(t *testing.T) {
	os.Setenv("TARGET_PORT", "notanumber")
	defer os.Unsetenv("TARGET_PORT")

	_, _, err := targetFromEnv()
	if err == nil {
		t.Error("expected error for non-numeric port, got nil")
	}
}

func TestTargetFromEnvPortOutOfRange(t *testing.T) {
	for _, port := range []string{"0", "65536", "-1"} {
		os.Setenv("TARGET_PORT", port)
		_, _, err := targetFromEnv()
		if err == nil {
			t.Errorf("expected error for port %q, got nil", port)
		}
		os.Unsetenv("TARGET_PORT")
	}
}

func TestTargetFromEnvIPv6(t *testing.T) {
	os.Setenv("TARGET_HOST", "::1")
	os.Setenv("TARGET_PORT", "8080")
	defer func() {
		os.Unsetenv("TARGET_HOST")
		os.Unsetenv("TARGET_PORT")
	}()

	addr, _, err := targetFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "[::1]:8080" {
		t.Errorf("expected '[::1]:8080', got %q", addr)
	}
}

func TestEnv(t *testing.T) {
	os.Setenv("EXISTING_VAR", "set_value")
	defer os.Unsetenv("EXISTING_VAR")

	tests := []struct {
		key      string
		fallback string
		expected string
	}{
		{"EXISTING_VAR", "fallback", "set_value"},
		{"MISSING_VAR", "fallback", "fallback"},
	}
	for _, tc := range tests {
		got := env(tc.key, tc.fallback)
		if got != tc.expected {
			t.Errorf("env(%q, %q) = %q, want %q", tc.key, tc.fallback, got, tc.expected)
		}
	}
}

func TestDurationEnv(t *testing.T) {
	os.Setenv("CUSTOM_DUR", "3s")
	defer os.Unsetenv("CUSTOM_DUR")

	if got := durationEnv("CUSTOM_DUR", 1); got != mustParseDuration(t, "3s") {
		t.Errorf("expected 3s, got %v", got)
	}

	if got := durationEnv("MISSING_DUR", 500*time.Millisecond); got != 500*time.Millisecond {
		t.Errorf("expected 500ms fallback, got %v", got)
	}

	os.Setenv("BAD_DUR", "not-a-duration")
	if got := durationEnv("BAD_DUR", 2*time.Second); got != 2*time.Second {
		t.Errorf("expected 2s fallback for invalid duration, got %v", got)
	}
	os.Unsetenv("BAD_DUR")
}

func TestProbeReachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, _ := ln.Accept()
			if conn != nil {
				conn.Close()
			}
			break
		}
	}()

	if !probe(ln.Addr().String(), 500*time.Millisecond) {
		t.Error("expected probe to succeed against listening port")
	}
}

func TestProbeUnreachable(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()

	if probe(addr, 200*time.Millisecond) {
		t.Error("expected probe to fail against closed port")
	}
}
