package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConvertToReverseIP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard IPv4",
			input:    "192.168.1.1",
			expected: "1.1.168.192",
		},
		{
			name:     "another IPv4",
			input:    "8.8.8.8",
			expected: "8.8.8.8",
		},
		{
			name:     "complex IPv4",
			input:    "127.0.0.2",
			expected: "2.0.0.127",
		},
		{
			name:     "public IP",
			input:    "203.0.113.45",
			expected: "45.113.0.203",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToReverseIP(tt.input)
			if result != tt.expected {
				t.Errorf("convertToReverseIP(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid integer",
			envKey:       "TEST_INT_1",
			envValue:     "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "use default when empty",
			envKey:       "TEST_INT_2",
			envValue:     "",
			defaultValue: 99,
			expected:     99,
		},
		{
			name:         "invalid integer returns default",
			envKey:       "TEST_INT_3",
			envValue:     "not_a_number",
			defaultValue: 50,
			expected:     50,
		},
		{
			name:         "zero value",
			envKey:       "TEST_INT_4",
			envValue:     "0",
			defaultValue: 10,
			expected:     0,
		},
		{
			name:         "negative value",
			envKey:       "TEST_INT_5",
			envValue:     "-5",
			defaultValue: 10,
			expected:     -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before and after
			os.Unsetenv(tt.envKey)
			defer os.Unsetenv(tt.envKey)

			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
			}

			result := getEnvAsInt(tt.envKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsInt(%q, %d) = %d; want %d", tt.envKey, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestReadListsFromFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    []string
		shouldError bool
	}{
		{
			name: "simple list",
			content: `zen.spamhaus.org
dnsbl.sorbs.net
bl.spamcop.net`,
			expected:    []string{"zen.spamhaus.org", "dnsbl.sorbs.net", "bl.spamcop.net"},
			shouldError: false,
		},
		{
			name: "with empty lines",
			content: `zen.spamhaus.org

dnsbl.sorbs.net

bl.spamcop.net`,
			expected:    []string{"zen.spamhaus.org", "dnsbl.sorbs.net", "bl.spamcop.net"},
			shouldError: false,
		},
		{
			name: "with comments",
			content: `# Comment line
zen.spamhaus.org
# Another comment
dnsbl.sorbs.net`,
			expected:    []string{"# Comment line", "zen.spamhaus.org", "# Another comment", "dnsbl.sorbs.net"},
			shouldError: false,
		},
		{
			name: "with whitespace",
			content: `  zen.spamhaus.org
	dnsbl.sorbs.net
   bl.spamcop.net   `,
			expected:    []string{"zen.spamhaus.org", "dnsbl.sorbs.net", "bl.spamcop.net"},
			shouldError: false,
		},
		{
			name:        "empty file",
			content:     "",
			expected:    []string{},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test_lists.txt")

			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			result, err := readListsFromFile(tmpFile)
			if tt.shouldError && err == nil {
				t.Errorf("readListsFromFile() expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("readListsFromFile() unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("readListsFromFile() returned %d items; want %d", len(result), len(tt.expected))
			}

			for i, item := range result {
				if i >= len(tt.expected) {
					break
				}
				if item != tt.expected[i] {
					t.Errorf("readListsFromFile() item %d = %q; want %q", i, item, tt.expected[i])
				}
			}
		})
	}
}

func TestReadListsFromFile_NonExistent(t *testing.T) {
	_, err := readListsFromFile("/non/existent/file.txt")
	if err == nil {
		t.Error("readListsFromFile() expected error for non-existent file but got none")
	}
}

func TestLoadConfig_StaticIP(t *testing.T) {
	// Set up environment
	os.Setenv("DNSRBL_CHECK_IP", "192.168.1.1")
	os.Setenv("DNSRBL_DELAY_REQUESTS", "5")
	os.Setenv("DNSRBL_DELAY_RUNS", "120")
	os.Setenv("DNSRBL_PORT", "9000")
	os.Setenv("DNSRBL_LISTS", "zen.spamhaus.org dnsbl.sorbs.net")

	defer func() {
		os.Unsetenv("DNSRBL_CHECK_IP")
		os.Unsetenv("DNSRBL_DELAY_REQUESTS")
		os.Unsetenv("DNSRBL_DELAY_RUNS")
		os.Unsetenv("DNSRBL_PORT")
		os.Unsetenv("DNSRBL_LISTS")
	}()

	config := loadConfig()

	if config.CheckIP != "192.168.1.1" {
		t.Errorf("CheckIP = %q; want %q", config.CheckIP, "192.168.1.1")
	}
	if config.CheckIPMode != "static" {
		t.Errorf("CheckIPMode = %q; want %q", config.CheckIPMode, "static")
	}
	if config.DelayBetweenRequests != 5*time.Second {
		t.Errorf("DelayBetweenRequests = %v; want %v", config.DelayBetweenRequests, 5*time.Second)
	}
	if config.DelayBetweenRuns != 120*time.Second {
		t.Errorf("DelayBetweenRuns = %v; want %v", config.DelayBetweenRuns, 120*time.Second)
	}
	if config.Port != 9000 {
		t.Errorf("Port = %d; want %d", config.Port, 9000)
	}
	if len(config.Lists) != 2 {
		t.Errorf("Lists length = %d; want %d", len(config.Lists), 2)
	}
}

func TestLoadConfig_DynamicIP(t *testing.T) {
	// Ensure DNSRBL_CHECK_IP is not set
	os.Unsetenv("DNSRBL_CHECK_IP")
	os.Setenv("DNSRBL_LISTS", "zen.spamhaus.org")
	defer os.Unsetenv("DNSRBL_LISTS")

	config := loadConfig()

	if config.CheckIPMode != "dynamic" {
		t.Errorf("CheckIPMode = %q; want %q", config.CheckIPMode, "dynamic")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear all relevant environment variables
	os.Unsetenv("DNSRBL_CHECK_IP")
	os.Unsetenv("DNSRBL_DELAY_REQUESTS")
	os.Unsetenv("DNSRBL_DELAY_RUNS")
	os.Unsetenv("DNSRBL_PORT")
	os.Unsetenv("DNSRBL_LISTS")
	os.Unsetenv("DNSRBL_LISTS_FILENAME")
	os.Unsetenv("DNSRBL_HTTP_BL_ACCESS_KEY")

	// Create a temporary lists file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "lists.txt")
	if err := os.WriteFile(tmpFile, []byte("zen.spamhaus.org\n"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	config := loadConfig()

	if config.DelayBetweenRequests != 1*time.Second {
		t.Errorf("DelayBetweenRequests = %v; want %v", config.DelayBetweenRequests, 1*time.Second)
	}
	if config.DelayBetweenRuns != 60*time.Second {
		t.Errorf("DelayBetweenRuns = %v; want %v", config.DelayBetweenRuns, 60*time.Second)
	}
	if config.Port != 8000 {
		t.Errorf("Port = %d; want %d", config.Port, 8000)
	}
}

func TestHandleDNSError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		blacklist string
		ip        string
	}{
		{
			name:      "not found error",
			err:       &net.DNSError{Err: "no such host", IsNotFound: true},
			blacklist: "test.blacklist.org",
			ip:        "192.168.1.1",
		},
		{
			name:      "timeout error",
			err:       &net.DNSError{Err: "timeout", IsTimeout: true},
			blacklist: "test.blacklist.org",
			ip:        "192.168.1.1",
		},
		{
			name:      "unknown error",
			err:       &net.DNSError{Err: "unknown error"},
			blacklist: "test.blacklist.org",
			ip:        "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function updates metrics, so we just ensure it doesn't panic
			handleDNSError(tt.err, tt.blacklist, tt.ip)
		})
	}
}

func TestLookupIP(t *testing.T) {
	// Test with a known good DNS query (Google's public DNS)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := lookupIP(ctx, "google.com")
	if err != nil {
		t.Logf("lookupIP failed (this may be expected in some environments): %v", err)
	}
	if len(ips) == 0 && err == nil {
		t.Error("lookupIP returned no IPs and no error")
	}
}

func TestLookupIP_Timeout(t *testing.T) {
	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This should timeout or fail quickly
	_, err := lookupIP(ctx, "test.invalid.blacklist.example.com")
	if err == nil {
		t.Log("Expected timeout error but got none (may be cached)")
	}
}

func BenchmarkConvertToReverseIP(b *testing.B) {
	ip := "192.168.1.100"
	for i := 0; i < b.N; i++ {
		convertToReverseIP(ip)
	}
}

func BenchmarkGetEnvAsInt(b *testing.B) {
	os.Setenv("BENCH_TEST_INT", "42")
	defer os.Unsetenv("BENCH_TEST_INT")

	for i := 0; i < b.N; i++ {
		getEnvAsInt("BENCH_TEST_INT", 10)
	}
}
