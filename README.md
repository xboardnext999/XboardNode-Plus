# XboardNode-Plus

Node backend for [Xboard](https://github.com/cedar2025/Xboard), based on [cedar2025/Xboard-Node](https://github.com/cedar2025/Xboard-Node). Supports `sing-box` / `xray-core` dual kernels.

> **Disclaimer**: This project is for educational and learning purposes only.

## Features

- Protocols: VMess, VLESS, Trojan, Shadowsocks, Hysteria/Hysteria2, TUIC, AnyTLS, Naive, SOCKS, HTTP, Mieru
- Sync: WebSocket push + REST polling dual channel
- User controls: speed limit, device limit, alive-IP tracking, hot update
- Deploy modes: node mode, machine mode, standalone mode
- Multi-instance: single process binding multiple panels / nodes
- Runtime sync safety fixes: see [FIXES.md](FIXES.md)

## Supported Protocols

The default kernel mode is `auto`. In this mode XboardNode-Plus chooses the effective kernel from the panel node configuration: `xhttp` / `splithttp` uses `xray`, and other supported transports use `singbox` by default. You can still force a kernel with `--kernel singbox` or `--kernel xray`.

| Kernel | Supported inbound protocols |
| --- | --- |
| `singbox` | `shadowsocks`, `vmess`, `vless`, `trojan`, `hysteria`, `hysteria2`, `tuic`, `anytls`, `naive`, `socks`, `http`, `mieru` |
| `xray` | `vmess`, `vless`, `trojan`, `shadowsocks`, `hysteria` |

Transport support:

| Kernel | Supported transports |
| --- | --- |
| `singbox` | `tcp`, `ws`, `grpc`, `httpupgrade`, `h2` / `http` |
| `xray` | `tcp`, `ws`, `grpc`, `httpupgrade`, `h2` / `http`, `xhttp` / `splithttp` |

Notes:

- `kernel.type: auto` keeps the config as an adaptive strategy. Runtime logs show both `configured_kernel` and `effective_kernel`.
- `hysteria`, `hysteria2`, `tuic`, and `anytls` require TLS certificate configuration.
- `trojan` requires TLS unless Reality is configured.
- Reality is supported for `vless` and `trojan`; it requires `tls_settings.private_key` and either `tls_settings.server_name` or `tls_settings.dest`.
- `custom_outbounds` can also use outbound-only protocols such as `wireguard`; those are not panel inbound deployment protocols.

## Install

### Docker

```bash
docker run -d --restart=always --network=host \
  -e apiHost=https://panel.com -e apiKey=TOKEN -e nodeID=1 \
  ghcr.io/xboardnext999/xboardnode-plus:latest
```

### Docker Compose

```bash
git clone --depth 1 https://github.com/xboardnext999/XboardNode-Plus.git
cd XboardNode-Plus
vim compose.yml   # set apiHost / apiKey / nodeID
docker compose up -d
```

### Installer (Linux systemd)

Use the release installer when `raw.githubusercontent.com` is rate-limited:

```bash
curl -fL https://github.com/xboardnext999/XboardNode-Plus/releases/latest/download/easy-install.sh | \
  sudo bash -s -- \
    --panel https://panel.example.com \
    --token TOKEN \
    --machine-id 1 \
    --domain node.example.com
```

This installs the service, downloads release binaries, configures nginx Trojan SNI passthrough, and creates a local camouflage site.

```bash
# Node mode
curl -fsSL https://raw.githubusercontent.com/xboardnext999/XboardNode-Plus/dev/install.sh | \
  sudo bash -s -- --mode node --panel https://panel.example.com --token TOKEN --node-id 1

# Machine mode
curl -fsSL https://raw.githubusercontent.com/xboardnext999/XboardNode-Plus/dev/install.sh | \
  sudo bash -s -- --mode machine --panel https://panel.example.com --token TOKEN --machine-id 1
```

The Linux installer stores configuration under `/etc/XboardNode-Plus`.
If an older `/etc/xboard-node` directory exists, the installer migrates it automatically.

### Replace Original Install Command

If a panel page still shows the original `cedar2025/xboard-node` install command, run this helper first:

```bash
curl -fsSL https://raw.githubusercontent.com/xboardnext999/XboardNode-Plus/dev/rewrite-install.sh | bash
```

Then paste the original install command. It will replace the installer URL with this repository and execute the converted command.

## xbctl

Run `xbctl` after installation for help. Common commands:

```bash
xbctl list                          # list all instances
xbctl status                        # running status
xbctl bind add-node --panel URL --token TOKEN --node-id 1
xbctl bind add-machine --panel URL --token TOKEN --machine-id 1
xbctl bind remove-node --panel URL --node-id 1
xbctl service restart
```

## Configuration

Legacy single-panel config is fully compatible. Appending bindings auto-migrates to `instances` format. See `config.yml.example`.

## Extensions

- Custom routes: [docs-custom-routes.md](docs-custom-routes.md)
- Custom outbounds: [docs-custom-outbounds.md](docs-custom-outbounds.md)
- DNS providers (ACME DNS-01): [docs-dns-providers.md](docs-dns-providers.md)
- Trojan nginx SNI passthrough: [docs-nginx-trojan.md](docs-nginx-trojan.md)

## License

MPL-2.0.
