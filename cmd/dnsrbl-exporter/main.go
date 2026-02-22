package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Version is set at build time via ldflags
var Version = "dev"

var (
	errorMapping = map[string]float64{
		"NXDOMAIN":        0,
		"Found":           1,
		"NoAnswer":        2,
		"NoNameservers":   3,
		"Timeout":         4,
		"Unknown":         5,
		"LifetimeTimeout": 4,
	}

	// Prometheus metrics
	dnsrblInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dnsrbl_info",
			Help: "General info about dnsrbl configuration",
		},
		[]string{"check_ip", "check_ip_mode", "delay_between_requests", "delay_between_runs"},
	)

	dnsrblTaskState = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "dnsrbl_task_state",
			Help: "Task state: 0=sleeping, 1=running",
		},
	)

	dnsrblListSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "dnsrbl_list_size",
			Help: "Number of blacklists active",
		},
	)

	dnsrblQuery = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dnsrbl_query",
			Help: "DNS queries",
		},
		[]string{"list", "ip", "result"},
	)

	dnsrblStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dnsrbl_status",
			Help: "DNSRBL check status: 0=ok, 1=found in blacklist, 2-5=error",
		},
		[]string{"list", "ip"},
	)

	httpblLastActivity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "httpbl_last_activity",
			Help: "ProjectHoneyPot.org last activity",
		},
		[]string{"list", "ip"},
	)

	httpblThreatScore = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "httpbl_threat_score",
			Help: "ProjectHoneyPot.org threat score",
		},
		[]string{"list", "ip"},
	)

	httpblVisitorType = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "httpbl_visitor_type",
			Help: "ProjectHoneyPot.org visitor type",
		},
		[]string{"list", "ip"},
	)

	requestDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Name: "request_processing_seconds",
			Help: "Time spent processing request",
		},
	)
)

// Config holds the application configuration
type Config struct {
	CheckIP              string
	CheckIPMode          string
	DelayBetweenRequests time.Duration
	DelayBetweenRuns     time.Duration
	Port                 int
	Lists                []string
	HTTPBLAccessKey      string
}

func main() {
	// Parse command-line flags
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("dnsrbl-exporter version %s\n", Version)
		os.Exit(0)
	}

	config := loadConfig()

	// Start Prometheus HTTP server
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		addr := fmt.Sprintf(":%d", config.Port)
		log.Printf("Starting HTTP server on %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Main loop
	for {
		checkIP := config.CheckIP
		if config.CheckIPMode == "dynamic" {
			var err error
			checkIP, err = getExternalIP()
			if err != nil {
				log.Printf("Error getting external IP: %v", err)
				time.Sleep(config.DelayBetweenRuns)
				continue
			}
		}

		log.Printf("Using %s as %s check IP", checkIP, config.CheckIPMode)
		dnsrblListSize.Set(float64(len(config.Lists)))
		log.Printf("Using %d blacklists", len(config.Lists))

		// Set info metric
		dnsrblInfo.WithLabelValues(
			checkIP,
			config.CheckIPMode,
			fmt.Sprintf("%ds", int(config.DelayBetweenRequests.Seconds())),
			fmt.Sprintf("%ds", int(config.DelayBetweenRuns.Seconds())),
		).Set(1)

		for _, blacklist := range config.Lists {
			if strings.HasPrefix(blacklist, "#") || strings.TrimSpace(blacklist) == "" {
				continue
			}

			dnsrblTaskState.Set(1) // running
			checkDNSRBL(checkIP, blacklist, config.HTTPBLAccessKey)
			dnsrblTaskState.Set(0) // sleeping

			log.Printf("Sleeping for %v...", config.DelayBetweenRequests)
			time.Sleep(config.DelayBetweenRequests)
		}

		log.Printf("Sleeping for %v...", config.DelayBetweenRuns)
		time.Sleep(config.DelayBetweenRuns)
	}
}

func loadConfig() *Config {
	config := &Config{
		DelayBetweenRequests: time.Duration(getEnvAsInt("DNSRBL_DELAY_REQUESTS", 1)) * time.Second,
		DelayBetweenRuns:     time.Duration(getEnvAsInt("DNSRBL_DELAY_RUNS", 60)) * time.Second,
		Port:                 getEnvAsInt("DNSRBL_PORT", 8000),
		HTTPBLAccessKey:      os.Getenv("DNSRBL_HTTP_BL_ACCESS_KEY"),
	}

	// Determine check IP mode
	if checkIP := os.Getenv("DNSRBL_CHECK_IP"); checkIP != "" {
		config.CheckIP = checkIP
		config.CheckIPMode = "static"
	} else {
		config.CheckIPMode = "dynamic"
	}

	// Load blacklist lists
	if lists := os.Getenv("DNSRBL_LISTS"); lists != "" {
		config.Lists = strings.Fields(lists)
	} else {
		filename := os.Getenv("DNSRBL_LISTS_FILENAME")
		if filename == "" {
			filename = "lists.txt"
		}
		var err error
		config.Lists, err = readListsFromFile(filename)
		if err != nil {
			log.Fatalf("Failed to read lists file: %v", err)
		}
	}

	return config
}

func readListsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lists []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lists = append(lists, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lists, nil
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func checkDNSRBL(ip, blacklist, httpblAccessKey string) {
	start := time.Now()
	defer func() {
		requestDuration.Observe(time.Since(start).Seconds())
	}()

	reverseIP := convertToReverseIP(ip)
	query := fmt.Sprintf("%s.%s.", reverseIP, blacklist)

	if blacklist == "dnsbl.httpbl.org" {
		if httpblAccessKey == "" {
			log.Printf("Skipping blacklist %s due to missing env DNSRBL_HTTP_BL_ACCESS_KEY", blacklist)
			return
		}
		query = fmt.Sprintf("%s.%s.%s.", httpblAccessKey, reverseIP, blacklist)
	}

	log.Printf("Checking %s.%s.", reverseIP, blacklist)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	answers, err := lookupIP(ctx, query)
	if err != nil {
		handleDNSError(err, blacklist, ip)
		return
	}

	if len(answers) == 0 {
		dnsrblQuery.WithLabelValues(blacklist, ip, "NoAnswer").Inc()
		dnsrblStatus.WithLabelValues(blacklist, ip).Set(errorMapping["NoAnswer"])
		log.Printf("Error: NoAnswer")
		return
	}

	for _, answer := range answers {
		result := answer.String()
		log.Printf("Match: %s found in %s", result, blacklist)

		if blacklist == "dnsbl.httpbl.org" {
			parts := strings.Split(result, ".")
			if len(parts) >= 4 {
				lastActivity, _ := strconv.ParseFloat(parts[1], 64)
				threatScore, _ := strconv.ParseFloat(parts[2], 64)
				visitorType, _ := strconv.ParseFloat(parts[3], 64)

				httpblLastActivity.WithLabelValues(blacklist, ip).Set(lastActivity)
				httpblThreatScore.WithLabelValues(blacklist, ip).Set(threatScore)
				httpblVisitorType.WithLabelValues(blacklist, ip).Set(visitorType)

				log.Printf("Last activity: %s days ago", parts[1])
				log.Printf("Threat score: %s", parts[2])
				log.Printf("Visitor type: %s", parts[3])
			}
		}
	}

	dnsrblQuery.WithLabelValues(blacklist, ip, "Found").Inc()
	dnsrblStatus.WithLabelValues(blacklist, ip).Set(errorMapping["Found"])
}

func lookupIP(ctx context.Context, query string) ([]net.IP, error) {
	resolver := &net.Resolver{}
	return resolver.LookupIP(ctx, "ip4", strings.TrimSuffix(query, "."))
}

func handleDNSError(err error, blacklist, ip string) {
	var errorType string

	if dnsErr, ok := err.(*net.DNSError); ok {
		if dnsErr.IsNotFound {
			errorType = "NXDOMAIN"
		} else if dnsErr.IsTimeout {
			errorType = "Timeout"
		} else {
			errorType = "Unknown"
		}
	} else {
		errorType = "Unknown"
	}

	log.Printf("Error: %s", errorType)
	dnsrblQuery.WithLabelValues(blacklist, ip, errorType).Inc()

	if val, ok := errorMapping[errorType]; ok {
		dnsrblStatus.WithLabelValues(blacklist, ip).Set(val)
	} else {
		dnsrblStatus.WithLabelValues(blacklist, ip).Set(errorMapping["Unknown"])
	}
}

func convertToReverseIP(ip string) string {
	parts := strings.Split(ip, ".")
	// Reverse the slice
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".")
}

func getExternalIP() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try multiple services for reliability
	services := []string{
		"https://api.ipify.org",
		"https://icanhazip.com",
		"https://ifconfig.me/ip",
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var lastErr error
	for _, service := range services {
		req, err := http.NewRequestWithContext(ctx, "GET", service, nil)
		if err != nil {
			lastErr = err
			continue
		}

		// Add headers to request plain text
		req.Header.Set("Accept", "text/plain")
		req.Header.Set("User-Agent", "dnsrbl-exporter/1.0")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			continue
		}

		ip := strings.TrimSpace(string(body))

		// Validate that we got an IP address, not HTML
		if net.ParseIP(ip) != nil {
			return ip, nil
		}

		lastErr = fmt.Errorf("invalid IP address received: %s", ip)
	}

	if lastErr != nil {
		return "", fmt.Errorf("failed to get external IP from all services: %w", lastErr)
	}
	return "", fmt.Errorf("failed to get external IP")
}
