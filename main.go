package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-sdk/client"
)

type NodeIP struct {
	IPv4 net.IP
	IPv6 net.IP
}

func main() {
	labelKey, ok := os.LookupEnv("HEADSCALE_DNS_LABEL_KEY")
	if !ok {
		labelKey = "headscale.dns.subdomain"
	}

	extraRecordsPath, ok := os.LookupEnv("HEADSCALE_DNS_JSON_PATH")
	if !ok {
		log.Fatal("HEADSCALE_DNS_JSON_PATH environment variable is required")
	}

	refreshSecondsStr, ok := os.LookupEnv("HEADSCALE_DNS_REFRESH_SECONDS")
	if !ok {
		refreshSecondsStr = "60"
	}
	refreshSeconds, err := strconv.Atoi(refreshSecondsStr)
	if err != nil {
		log.Fatalf("Invalid HEADSCALE_DNS_REFRESH_SECONDS value: %v", err)
	}

	baseDomain, ok := os.LookupEnv("HEADSCALE_DNS_BASE_DOMAIN")
	if !ok {
		baseDomain = "ts.net"
	}

	nodeHostname, ok := os.LookupEnv("HEADSCALE_DNS_NODE_HOSTNAME")
	if !ok {
		log.Fatal("HEADSCALE_DNS_NODE_HOSTNAME environment variable is required")
	}

	nodeIP4Str, ok := os.LookupEnv("HEADSCALE_DNS_NODE_IP")
	if !ok {
		log.Fatal("HEADSCALE_DNS_NODE_IP environment variable is required. Use tailscale ip to get the node IP.")
	}

	nodeIP6Str, ok := os.LookupEnv("HEADSCALE_DNS_NODE_IP6")
	if !ok {
		log.Printf("HEADSCALE_DNS_NODE_IP6 environment variable is required. AAAA records will not be created.")
	}

	// Parse IP addresses
	nodeConfig := NodeIP{}

	nodeConfig.IPv4 = net.ParseIP(nodeIP4Str)
	if nodeConfig.IPv4 == nil {
		log.Fatal("Invalid IPv4 address provided in HEADSCALE_DNS_NODE_IP")
	}

	if nodeIP6Str != "" {
		nodeConfig.IPv6 = net.ParseIP(nodeIP6Str)
		if nodeConfig.IPv6 == nil {
			log.Fatal("Invalid IPv6 address provided in HEADSCALE_DNS_NODE_IP6")
		}
	}

	ctx := context.Background()
	// Create client with options
	cli, err := client.New(
		ctx,
		getDockerOptions()...,
	)
	if err != nil {
		panic(err)
	}

	// Cleanup
	defer func() {
		err := cli.Close()
		if err != nil {
			log.Fatalf("Error closing Docker client: %v", err)
		}
	}()

	log.Printf("Using settings: ")
	log.Printf(" - Label Key: %s", labelKey)
	log.Printf(" - JSON extra_records_path path: %s", extraRecordsPath)
	log.Printf(" - Base Domain: %s", baseDomain)
	log.Printf(" - Node Hostname: %s", nodeHostname)
	log.Printf(" - Node IPv4 Address: %s", nodeConfig.IPv4.String())
	if nodeConfig.IPv6 != nil {
		log.Printf(" - Node IPv6 Address: %s", nodeConfig.IPv6.String())
	}
	ipInfo := nodeConfig.IPv4.String()
	if nodeConfig.IPv6 != nil {
		ipInfo += "/" + nodeConfig.IPv6.String()
	}
	log.Printf(" - Example URL: service.%s.%s -> %s", nodeHostname, baseDomain, ipInfo)
	log.Printf(" - Refresh Interval: %d seconds", refreshSeconds)

	processContainers := func() {
		containersRunning, err := getRunningDockerContainers(cli, ctx)
		if err != nil {
			log.Fatalf("Error getting running containers: %v", err)
			return
		}
		log.Printf("Found %d running containers", len(containersRunning))

		subdomains, err := getSubdomainsFromLabels(containersRunning, labelKey)
		if err != nil {
			log.Fatalf("Error getting hostnames: %v", err)
			return
		}
		log.Printf("Discovered %d subdomains\n", len(subdomains))

		// Create JSON records
		records := createJSON(subdomains, nodeHostname+"."+baseDomain, nodeConfig)

		// Marshal to JSON with proper formatting
		jsonData, err := json.MarshalIndent(sortJSON(records), "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling JSON: %v", err)
			return
		}

		// Write to file
		err = os.WriteFile(extraRecordsPath, jsonData, 0o644)
		if err != nil {
			log.Fatalf("Error writing JSON file: %v", err)
			return
		}
		log.Printf("Successfully wrote %d DNS records to JSON file", len(records))
	}

	// Run once on startup
	processContainers()

	// Repeat every HEADSCALE_DNS_REFRESH_SECONDS seconds
	ticker := time.NewTicker(time.Duration(refreshSeconds) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		processContainers()
	}
}

// https://github.com/juanfont/headscale/blob/main/docs/ref/dns.md
func createJSON(subdomains []string, domain string, nodeConfig NodeIP) []map[string]any {
	records := make([]map[string]any, 0)

	for _, subdomain := range subdomains {
		// Create A record for IPv4
		if nodeConfig.IPv4 != nil {
			record := map[string]any{
				"name":  subdomain + "." + domain,
				"type":  "A",
				"value": nodeConfig.IPv4.String(),
			}
			records = append(records, record)
		}

		// Create AAAA record for IPv6 if available
		if nodeConfig.IPv6 != nil {
			record := map[string]any{
				"name":  subdomain + "." + domain,
				"type":  "AAAA",
				"value": nodeConfig.IPv6.String(),
			}
			records = append(records, record)
		}
	}

	return records
}

func sortJSON(records []map[string]any) []map[string]any {
	// Sort the keys
	// "Be sure to "sort keys" and produce a stable output in case you generate the JSON file with a script.
	// Headscale uses a checksum to detect changes to the file and a stable output avoids unnecessary processing."
	sort.Slice(records, func(i, j int) bool {
		nameI := records[i]["name"].(string)
		nameJ := records[j]["name"].(string)

		if nameI != nameJ {
			return nameI < nameJ
		}

		typeI := records[i]["type"].(string)
		typeJ := records[j]["type"].(string)
		return typeI < typeJ
	})

	return records
}

func getDockerOptions() []client.ClientOption {
	options := []client.ClientOption{}

	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost != "" {
		log.Printf("Using DOCKER_HOST: %s", dockerHost)
		options = append(options, client.WithDockerHost(dockerHost))
	}

	dockerContext := os.Getenv("DOCKER_CONTEXT")
	if dockerContext != "" {
		log.Printf("Using DOCKER_CONTEXT: %s", dockerContext)
		options = append(options, client.WithDockerContext(dockerContext))
	}

	return options
}

func getRunningDockerContainers(cli client.SDKClient, ctx context.Context) ([]container.Summary, error) {
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		log.Fatalf("Error listing containers: %v", err)
		return nil, err
	}

	containersRunning := []container.Summary{}

	for _, container := range containers {
		if container.State == "running" {
			containersRunning = append(containersRunning, container)
		}
	}

	return containersRunning, nil
}

func getSubdomainsFromLabels(containers []container.Summary, labelKey string) ([]string, error) {
	hostnames := []string{}

	for _, container := range containers {
		if labelValue, ok := container.Labels[labelKey]; ok {
			// Split the label value by | to support multiple hostnames
			splitHostnames := strings.SplitSeq(labelValue, "|")
			for hostname := range splitHostnames {
				if trimmedHostname := strings.TrimSpace(hostname); trimmedHostname != "" {
					hostnames = append(hostnames, trimmedHostname)
				}
			}
		}
	}

	return hostnames, nil
}
