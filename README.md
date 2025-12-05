# Headscale DNS Docker

Automatically generates DNS records for Headscale `extra_records.json` based on Docker container labels.
It is meant to supplement Docker label based reverse proxy setups (e.g., Traefik, Nginx Proxy Manager, etc.) when using Headscale as Tailscale control server.

Refer: <https://github.com/juanfont/headscale/blob/main/docs/ref/dns.md>

## Quick Start

### Using Pre-built Image from GHCR

1. Create a `docker-compose.yml` file (or use the provided one)
2. Update environment variables with your settings
3. Run: `docker compose up -d`

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HEADSCALE_DNS_JSON_PATH` | Yes | - | Path to write the extra_records.json file |
| `HEADSCALE_DNS_NODE_HOSTNAME` | Yes | - | Hostname of the node running the containers |
| `HEADSCALE_DNS_NODE_IP` | Yes | - | IPv4 address of the node |
| `HEADSCALE_DNS_NODE_IP6` | No | - | IPv6 address of the node |
| `HEADSCALE_DNS_BASE_DOMAIN` | No | `ts.net` | Base domain for DNS records |
| `HEADSCALE_DNS_LABEL_KEY` | No | `headscale.dns.subdomain` | Docker label key to look for (in seconds) |
| `HEADSCALE_DNS_REFRESH_SECONDS` | No | `60` | How often to scan containers |
| `HEADSCALE_DNS_NO_BASE_DOMAIN` | No | `false` | Create additional records without base domain |
| `DOCKER_HOST` | No | `unix:///var/run/docker.sock` | Docker host socket path |
| `DOCKER_CONTEXT` | No | - | Docker Context |

### Deployment Example with Docker Compose

```yaml
services:
  headscale-dns-docker:
    image: ghcr.io/pranaovs/headscale-dns-docker:latest
    container_name: headscale-dns-docker
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /var/lib/headscale:/data # Headscale extra_records.json directory
    environment:
      - HEADSCALE_DNS_JSON_PATH=/data/extra_records.json
      - HEADSCALE_DNS_NODE_HOSTNAME=<Tailscale Hostname> # tailscale whois $(tailscale ip --1)
      - HEADSCALE_DNS_NODE_IP=<Tailscale IPv4>
      - HEADSCALE_DNS_NODE_IP6=<Tailscale IPv6>
      - HEADSCALE_DNS_BASE_DOMAIN=ts.net
      # - HEADSCALE_DNS_LABEL_KEY=headscale.dns.subdomain
      # - HEADSCALE_DNS_REFRESH_SECONDS=60
      - HEADSCALE_DNS_NO_BASE_DOMAIN=true # Create additional DNS Records without HEADSCALE_DNS_BASE_DOMAIN
```

### Usage

1. Label your containers:

```yaml
services:
  myapp:
    image: myapp
    labels:
      - "traefik.http.routers.myapp.rule=Host(`myapp.your-node-hostname.ts.net`) || Host(`myapp.your-node-hostname`)"
      - "headscale.dns.subdomain=myapp"
```

2. A DNS record(s) will be created for `myapp.your-node-hostname.ts.net` -> `HEADSCALE_DNS_NODE_IP` (and `HEADSCALE_DNS_NODE_IP6` if set).
3. If `HEADSCALE_DNS_NO_BASE_DOMAIN` is set to `true`, an additional record for `myapp.your-node-hostname` -> `HEADSCALE_DNS_NODE_IP` (and `HEADSCALE_DNS_NODE_IP6` if set) will be created.

## Building from Source

```bash
docker build -t headscale-dns-docker .
```

## GitHub Actions

This repository automatically builds and pushes Docker images to GitHub Container Registry on:

- Every push to `main` branch (tagged as `latest`)
- Every tagged release (tagged as version numbers)

The image is available at: `ghcr.io/pranaovs/headscale-dns-docker:latest`

---

_Disclaimer: README.md, Dockerfile and .github/ created using Claude Sonnet 4.5 (GitHub Copilot). Please report any problems/inconsistencies if found._
