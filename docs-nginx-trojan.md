# Trojan + nginx SNI passthrough

This guide is for a Trojan node that needs to share the public `443` port with
a normal HTTPS site. nginx listens on `443`, reads the TLS SNI with
`ssl_preread`, and forwards raw TCP to either the Trojan inbound or the web
upstream. nginx does not terminate TLS in this mode.

If the Trojan client and the browser both use the same SNI/domain, nginx cannot
distinguish them with `ssl_preread`. Use the "Same-domain camouflage" section
below: nginx routes that SNI to Trojan, and the Trojan inbound falls back
invalid/browser requests to a local HTTP camouflage site.

## Panel/node settings

Use a local-only Trojan port so the public entry is nginx:

- Protocol: `trojan`
- TLS: enabled, or Reality if that is your chosen Trojan mode
- Server port: for example `10443`
- Listen IP: `127.0.0.1`
- Certificate: configure as usual for the Trojan domain

The web upstream must also speak HTTPS, for example Caddy/nginx/Apache on
`127.0.0.1:8443`. A plain HTTP upstream will not work with this stream
passthrough layout because nginx forwards the original TLS bytes.

## Generate nginx config

Preview a complete top-level `stream {}` config:

```bash
xbctl nginx trojan-sni \
  --domain trojan.example.com \
  --trojan-upstream 127.0.0.1:10443 \
  --web-upstream 127.0.0.1:8443
```

If your `/etc/nginx/nginx.conf` already includes files inside a top-level
`stream {}` block, generate only the inner snippet:

```bash
xbctl nginx trojan-sni \
  --format snippet \
  --domain trojan.example.com \
  --trojan-upstream 127.0.0.1:10443 \
  --web-upstream 127.0.0.1:8443 \
  --write --force --test --reload
```

The default write path is:

```text
/etc/nginx/stream.d/xboard-node-trojan-sni.conf
```

Make sure nginx includes that directory at top level:

```nginx
stream {
    include /etc/nginx/stream.d/*.conf;
}
```

If the system uses a different include layout, use `--output` and place the
file where your nginx build expects stream snippets.

## Optional real client IP

You can pass the PROXY protocol through nginx:

```bash
xbctl nginx trojan-sni ... --proxy-protocol
```

Only enable this when the panel/node is also configured to accept PROXY
protocol, otherwise the Trojan inbound will reject the connection.

## Same-domain camouflage

Use this when `https://sg3.oone.us` should look like a normal website, while
Trojan clients also use `sg3.oone.us` as their SNI.

The flow is:

```text
Internet :443
  -> nginx stream ssl_preread
  -> Trojan inbound 127.0.0.1:10443
  -> browser/invalid Trojan request fallback
  -> local HTTP camouflage site 127.0.0.1:8080
```

Generate a random local camouflage website:

```bash
xbctl nginx camouflage-site \
  --domain sg3.oone.us \
  --listen 127.0.0.1:8080 \
  --write --force --test --reload
```

Available templates:

```text
random, cloud, studio, docs, commerce, status
```

To pin one template instead of random:

```bash
xbctl nginx camouflage-site \
  --domain sg3.oone.us \
  --template cloud \
  --write --force --test --reload
```

Configure the Trojan node with a local-only TLS inbound, for example:

- Protocol: `trojan`
- Server port: `10443`
- Listen IP: `127.0.0.1`
- TLS: enabled
- SNI/server name: `sg3.oone.us`
- TLS settings:

```yaml
fallback:
  server: 127.0.0.1
  server_port: 8080
```

The shorthand also works:

```yaml
fallback: 127.0.0.1:8080
```

Then route the public SNI to Trojan:

```bash
xbctl nginx trojan-sni \
  --domain sg3.oone.us \
  --trojan-upstream 127.0.0.1:10443 \
  --web-upstream 127.0.0.1:8443 \
  --write --force --test --reload
```

`--web-upstream` is still an HTTPS upstream for unmatched or explicitly mapped
SNI. It is not used for browser traffic that uses the Trojan domain itself;
same-domain browser traffic is handled by Trojan fallback and therefore uses
the HTTP camouflage site above.
