package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	address, timeout, err := targetFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	mode := "wait" // default
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "hold":
		// Long-lived container process. Docker HEALTHCHECK runs separately.
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig

	case "probe", "daemon":
		// One probe attempt; intended for Docker HEALTHCHECK.
		if probe(address, timeout) {
			os.Exit(0)
		}
		os.Exit(1)

	default:
		// One-shot wait mode: poll until reachable (or max wait exceeded), then exit 0/1.
		interval := durationEnv("INTERVAL", 250*time.Millisecond)
		maxWait := durationEnv("MAX_WAIT", 30*time.Second)
		if maxWait <= 0 {
			maxWait = 30 * time.Second
		}
		if interval <= 0 {
			interval = 250 * time.Millisecond
		}

		deadline := time.Now().Add(maxWait)
		for time.Now().Before(deadline) {
			if probe(address, timeout) {
				fmt.Println("target reachable:", address)
				os.Exit(0)
			}
			time.Sleep(interval)
		}
		fmt.Fprintf(os.Stderr, "timeout waiting for %s\n", address)
		os.Exit(1)
	}
}

func probe(address string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func targetFromEnv() (address string, timeout time.Duration, err error) {
	host := env("TARGET_HOST", "localhost")
	port := env("TARGET_PORT", "80")
	// Port must be a valid numeric TCP port.
	p, convErr := strconv.Atoi(port)
	if convErr != nil {
		return "", 0, fmt.Errorf("TARGET_PORT must be a number, got %q: %w", port, convErr)
	}
	if p <= 0 || p > 65535 {
		return "", 0, errors.New("TARGET_PORT must be between 1 and 65535")
	}
	timeout = durationEnv("TIMEOUT", 2*time.Second)
	if timeout <= 0 {
		return "", 0, errors.New("TIMEOUT must be a positive duration (e.g. 2s)")
	}
	address = net.JoinHostPort(host, port)
	return address, timeout, nil
}
