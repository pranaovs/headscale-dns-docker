package main

import (
	"context"
	"log"
	"os"

	"github.com/docker/go-sdk/client"
)

func main() {
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
