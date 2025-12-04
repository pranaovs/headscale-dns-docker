package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-sdk/client"
)

func main() {
	ctx := context.Background()

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

	fmt.Printf("Checking %d containers:\n\n", len(containers))

	containersRunning := []container.Summary{}

	for _, container := range containers {
		if container.State == "running" {
			containersRunning = append(containersRunning, container)
		}
	}

	return containersRunning, nil
}

func getHostnames(containers []container.Summary, labelKey string) ([]string, error) {
	hostnames := []string{}

	for _, container := range containers {
		if labelValue, ok := container.Labels[labelKey]; ok {
			hostnames = append(hostnames, labelValue)
		}
	}

	return hostnames, nil
}
