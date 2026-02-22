package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	file, err := os.Open("lists.txt")
	if err != nil {
		fmt.Printf("Error opening lists.txt: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	var working []string
	var notWorking []string
	var total int

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		total++
		fmt.Printf("Testing %s... ", line)

		if checkDNSBL(line) {
			fmt.Println("✓ OK")
			working = append(working, line)
		} else {
			fmt.Println("✗ FAILED")
			notWorking = append(notWorking, line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Printf("Total lists: %d\n", total)
	fmt.Printf("Working: %d\n", len(working))
	fmt.Printf("Not working: %d\n", len(notWorking))

	if len(notWorking) > 0 {
		fmt.Println("\n=== Lists that are NOT working ===")
		for _, list := range notWorking {
			fmt.Printf("  - %s\n", list)
		}
	}
}

func checkDNSBL(blacklist string) bool {
	// Validate input
	if blacklist == "" {
		return false
	}

	// Use 127.0.0.2 as test IP - should return NXDOMAIN for most lists
	// We're just checking if the DNS server responds
	testQuery := fmt.Sprintf("2.0.0.127.%s", blacklist)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolver := &net.Resolver{}

	// Try to resolve - we don't care about the result, just that we get a DNS response
	_, err := resolver.LookupIP(ctx, "ip4", testQuery)

	// NXDOMAIN (not found) is actually a good response - it means the DNS server is working
	if err != nil {
		if dnsErr, ok := err.(*net.DNSError); ok {
			// NXDOMAIN means the server responded (good)
			if dnsErr.IsNotFound {
				return true
			}
			// Timeout or other network errors mean the server is not responding
			if dnsErr.IsTimeout || dnsErr.IsTemporary {
				return false
			}
		}
		// For other errors, the server is likely not working
		return false
	}

	// Got an IP response - that's also good
	return true
}
