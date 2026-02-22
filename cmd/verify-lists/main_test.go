package main

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestCheckDNSBL tests the checkDNSBL function with various scenarios
func TestCheckDNSBL(t *testing.T) {
	tests := []struct {
		name      string
		blacklist string
		wantOk    bool
		skip      bool
		skipMsg   string
	}{
		{
			name:      "Valid DNS server with NXDOMAIN response",
			blacklist: "zen.spamhaus.org",
			wantOk:    true,
			skip:      false,
		},
		{
			name:      "Reserved .invalid TLD (returns NXDOMAIN but DNS works)",
			blacklist: "this-definitely-does-not-exist-12345.invalid",
			wantOk:    true, // NXDOMAIN means DNS is responding
			skip:      false,
		},
		{
			name:      "Empty blacklist",
			blacklist: "",
			wantOk:    false,
			skip:      false,
		},
		{
			name:      "Another well-known RBL",
			blacklist: "bl.spamcop.net",
			wantOk:    true,
			skip:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip(tt.skipMsg)
			}

			// Add timeout to prevent hanging tests
			done := make(chan bool, 1)
			var got bool

			go func() {
				got = checkDNSBL(tt.blacklist)
				done <- true
			}()

			select {
			case <-done:
				if got != tt.wantOk {
					t.Errorf("checkDNSBL(%q) = %v, want %v", tt.blacklist, got, tt.wantOk)
				}
			case <-time.After(10 * time.Second):
				t.Errorf("checkDNSBL(%q) timed out", tt.blacklist)
			}
		})
	}
}

// TestCheckDNSBLInputValidation tests input validation
func TestCheckDNSBLInputValidation(t *testing.T) {
	tests := []struct {
		name      string
		blacklist string
		wantOk    bool
	}{
		{
			name:      "Empty string",
			blacklist: "",
			wantOk:    false,
		},
		{
			name:      "Valid domain",
			blacklist: "zen.spamhaus.org",
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkDNSBL(tt.blacklist)
			if got != tt.wantOk {
				t.Errorf("checkDNSBL(%q) = %v, want %v", tt.blacklist, got, tt.wantOk)
			}
		})
	}
}

// TestDNSResolution tests basic DNS resolution functionality
func TestDNSResolution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolver := &net.Resolver{}

	// Test resolving a well-known domain
	_, err := resolver.LookupIP(ctx, "ip4", "google.com")
	if err != nil {
		t.Skip("Network connectivity issue - skipping DNS resolution test")
	}
}

// BenchmarkCheckDNSBL benchmarks the checkDNSBL function
func BenchmarkCheckDNSBL(b *testing.B) {
	blacklist := "zen.spamhaus.org"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checkDNSBL(blacklist)
	}
}

// BenchmarkCheckDNSBLInvalid benchmarks with an invalid blacklist
func BenchmarkCheckDNSBLInvalid(b *testing.B) {
	blacklist := "nonexistent.invalid"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checkDNSBL(blacklist)
	}
}
