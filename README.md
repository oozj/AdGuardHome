# AdGuard Home with Upstream DNS Rule Groups

This is an AdGuard Home fork that adds ordered upstream DNS rule groups while
keeping the familiar AdGuard Home interface.

## Features

- Remote hostname-rule subscriptions with manual refresh.
- Custom rule groups with editable DNS servers and a full-page rule editor.
- Ordered matching by priority, with the list order matching execution order.
- Exception rules that skip the current group and continue to the next group.
- A non-removable Default group for unmatched DNS questions.
- Hostname match testing and visible subscription or compile errors.

Matching is based on DNS question names.  URL paths, query strings, and
fragments are not available in DNS requests and are therefore ignored.

## Container images

Images are published to GitHub Container Registry:

```text
ghcr.io/oozj/adguardhome:latest
ghcr.io/oozj/adguardhome:0.107.79-custom.1
```

Use `latest` for the newest build or the fixed version tag for reproducible
deployments.

## Docker deployment

Create persistent directories:

```sh
sudo mkdir -p /opt/adguardhome/conf /opt/adguardhome/work
```

Start the container with host networking:

```sh
docker run -d \
  --name adguardhome \
  --restart unless-stopped \
  --network host \
  -v /opt/adguardhome/conf:/opt/adguardhome/conf \
  -v /opt/adguardhome/work:/opt/adguardhome/work \
  ghcr.io/oozj/adguardhome:latest
```

Open the setup or management interface:

```text
http://SERVER_IP:3000
```

The DNS service listens on TCP and UDP port 53.  Ensure those ports and TCP
port 3000 are not already used by another service.

## Docker Compose

Create `compose.yaml`:

```yaml
services:
  adguardhome:
    image: ghcr.io/oozj/adguardhome:latest
    container_name: adguardhome
    network_mode: host
    restart: unless-stopped
    volumes:
      - /opt/adguardhome/conf:/opt/adguardhome/conf
      - /opt/adguardhome/work:/opt/adguardhome/work
```

Start it:

```sh
docker compose up -d
```

## Upgrade

For Docker Compose:

```sh
docker compose pull
docker compose up -d
```

For `docker run`, pull the image, remove the old container, and run the same
deployment command again.  Configuration and runtime data remain in the two
mounted directories.

## Rule evaluation order

1. Client-specific upstream DNS settings.
2. Enabled subscription and custom groups from top to bottom.
3. The Default group.

Lower numeric priority values appear earlier.  Groups with the same priority
retain their saved order.

## License

This fork follows the license of the upstream
[AdGuard Home](https://github.com/AdguardTeam/AdGuardHome) project.
