package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultNginxTrojanSNIPath = "/etc/nginx/stream.d/xboard-node-trojan-sni.conf"

type nginxTrojanSNIOptions struct {
	Domain         string
	WebDomain      string
	TrojanUpstream string
	WebUpstream    string
	Listens        []string
	DefaultBackend string
	Format         string
	Output         string
	Write          bool
	Force          bool
	Test           bool
	Reload         bool
	ProxyProtocol  bool
}

func runNginx(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: xbctl nginx <trojan-sni|camouflage-site>")
	}
	switch args[0] {
	case "camouflage-site":
		return runNginxCamouflageSite(args[1:])
	case "trojan-sni":
		return runNginxTrojanSNI(args[1:])
	default:
		return fmt.Errorf("unknown nginx command: %s", args[0])
	}
}

func runNginxTrojanSNI(args []string) error {
	opts := nginxTrojanSNIOptions{
		TrojanUpstream: "127.0.0.1:10443",
		WebUpstream:    "127.0.0.1:8443",
		Listens:        []string{"443"},
		DefaultBackend: "web",
		Format:         "full",
		Output:         "-",
	}

	listenOverridden := false
	formatOverridden := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--domain", "--trojan-domain":
			if i+1 >= len(args) {
				return errors.New("missing value for --domain")
			}
			i++
			opts.Domain = args[i]
		case "--web-domain":
			if i+1 >= len(args) {
				return errors.New("missing value for --web-domain")
			}
			i++
			opts.WebDomain = args[i]
		case "--trojan-upstream":
			if i+1 >= len(args) {
				return errors.New("missing value for --trojan-upstream")
			}
			i++
			opts.TrojanUpstream = args[i]
		case "--web-upstream":
			if i+1 >= len(args) {
				return errors.New("missing value for --web-upstream")
			}
			i++
			opts.WebUpstream = args[i]
		case "--listen":
			if i+1 >= len(args) {
				return errors.New("missing value for --listen")
			}
			i++
			if !listenOverridden {
				opts.Listens = nil
				listenOverridden = true
			}
			opts.Listens = append(opts.Listens, args[i])
		case "--default":
			if i+1 >= len(args) {
				return errors.New("missing value for --default")
			}
			i++
			opts.DefaultBackend = args[i]
		case "--format":
			if i+1 >= len(args) {
				return errors.New("missing value for --format")
			}
			i++
			opts.Format = args[i]
			formatOverridden = true
		case "--output", "-o":
			if i+1 >= len(args) {
				return errors.New("missing value for --output")
			}
			i++
			opts.Output = args[i]
		case "--write":
			opts.Write = true
			if opts.Output == "-" {
				opts.Output = defaultNginxTrojanSNIPath
			}
		case "--force":
			opts.Force = true
		case "--test":
			opts.Test = true
		case "--reload":
			opts.Reload = true
		case "--proxy-protocol":
			opts.ProxyProtocol = true
		case "--help", "-h":
			printNginxTrojanSNIUsage()
			return nil
		default:
			return fmt.Errorf("unknown trojan-sni arg: %s", args[i])
		}
	}
	if opts.Write && !formatOverridden {
		opts.Format = "snippet"
	}

	conf, err := renderNginxTrojanSNI(opts)
	if err != nil {
		return err
	}

	if opts.Write {
		if err := writeNginxConfig(opts.Output, conf, opts.Force); err != nil {
			return err
		}
		fmt.Printf("Wrote %s\n", opts.Output)
	} else {
		fmt.Print(conf)
	}

	if opts.Test {
		if err := runPrivilegedCommand("nginx", "-t"); err != nil {
			return err
		}
	}
	if opts.Reload {
		if err := runPrivilegedCommand("nginx", "-s", "reload"); err != nil {
			return err
		}
	}
	return nil
}

func printNginxTrojanSNIUsage() {
	fmt.Println(`xbctl nginx trojan-sni [flags]

Generate nginx stream/ssl_preread config for sharing one 443 port between a
Trojan TLS inbound and a normal HTTPS site.

Required:
  --domain DOMAIN              SNI domain routed to Trojan

Optional:
  --trojan-upstream HOST:PORT  local Trojan TLS inbound (default 127.0.0.1:10443)
  --web-upstream HOST:PORT     local HTTPS web upstream (default 127.0.0.1:8443)
  --web-domain DOMAIN          explicit SNI domain routed to web (default uses nginx map default)
  --listen ADDR                nginx listen value; repeatable (default 443)
  --default web|trojan         backend for unmatched SNI (default web)
  --format full|snippet        include outer stream {} block or not (default full)
  --proxy-protocol             pass PROXY protocol to upstreams
  --output PATH, -o PATH       output path when --write is used
  --write                      write config to /etc/nginx/stream.d/xboard-node-trojan-sni.conf
  --force                      overwrite existing output
  --test                       run nginx -t after output/write
  --reload                     run nginx -s reload after output/write`)
}

func renderNginxTrojanSNI(opts nginxTrojanSNIOptions) (string, error) {
	opts.Domain = strings.TrimSpace(opts.Domain)
	if opts.Domain == "" {
		return "", errors.New("--domain is required")
	}
	if opts.TrojanUpstream == "" {
		return "", errors.New("--trojan-upstream is required")
	}
	if opts.WebUpstream == "" {
		return "", errors.New("--web-upstream is required")
	}
	if len(opts.Listens) == 0 {
		return "", errors.New("at least one --listen is required")
	}
	if opts.DefaultBackend == "" {
		opts.DefaultBackend = "web"
	}
	if opts.Format == "" {
		opts.Format = "full"
	}
	if opts.DefaultBackend != "web" && opts.DefaultBackend != "trojan" {
		return "", errors.New("--default must be web or trojan")
	}
	if opts.Format != "full" && opts.Format != "snippet" {
		return "", errors.New("--format must be full or snippet")
	}

	values := map[string]string{
		"domain":          opts.Domain,
		"trojan-upstream": opts.TrojanUpstream,
		"web-upstream":    opts.WebUpstream,
	}
	if opts.WebDomain != "" {
		values["web-domain"] = opts.WebDomain
	}
	for field, value := range values {
		if err := validateNginxToken(field, value); err != nil {
			return "", err
		}
	}
	for _, listen := range opts.Listens {
		if err := validateNginxToken("listen", listen); err != nil {
			return "", err
		}
	}

	defaultBackend := "xboard_node_web"
	if opts.DefaultBackend == "trojan" {
		defaultBackend = "xboard_node_trojan"
	}

	var body strings.Builder
	body.WriteString("# Generated by xbctl nginx trojan-sni.\n")
	body.WriteString("# Trojan keeps TLS enabled on the upstream; nginx only reads SNI and passes raw TCP.\n")
	body.WriteString("# If this file is included inside an existing top-level stream {} block, generate with --format snippet.\n\n")
	body.WriteString("map $ssl_preread_server_name $xboard_node_backend {\n")
	fmt.Fprintf(&body, "    %s xboard_node_trojan;\n", opts.Domain)
	if opts.WebDomain != "" {
		fmt.Fprintf(&body, "    %s xboard_node_web;\n", opts.WebDomain)
	}
	fmt.Fprintf(&body, "    default %s;\n", defaultBackend)
	body.WriteString("}\n\n")
	fmt.Fprintf(&body, "upstream xboard_node_trojan {\n    server %s;\n}\n\n", opts.TrojanUpstream)
	fmt.Fprintf(&body, "upstream xboard_node_web {\n    server %s;\n}\n\n", opts.WebUpstream)
	body.WriteString("server {\n")
	for _, listen := range opts.Listens {
		fmt.Fprintf(&body, "    listen %s reuseport;\n", listen)
	}
	body.WriteString("    proxy_pass $xboard_node_backend;\n")
	body.WriteString("    ssl_preread on;\n")
	body.WriteString("    proxy_connect_timeout 10s;\n")
	body.WriteString("    proxy_timeout 1h;\n")
	if opts.ProxyProtocol {
		body.WriteString("    proxy_protocol on;\n")
	}
	body.WriteString("}\n")

	if opts.Format == "snippet" {
		return body.String(), nil
	}

	var full strings.Builder
	full.WriteString("stream {\n")
	for _, line := range strings.Split(strings.TrimRight(body.String(), "\n"), "\n") {
		if line == "" {
			full.WriteString("\n")
			continue
		}
		full.WriteString("    ")
		full.WriteString(line)
		full.WriteString("\n")
	}
	full.WriteString("}\n")
	return full.String(), nil
}

func validateNginxToken(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s cannot be empty", field)
	}
	if strings.ContainsAny(value, "{};\r\n\t ") {
		return fmt.Errorf("%s contains characters that are unsafe for nginx config: %q", field, value)
	}
	return nil
}

func writeNginxConfig(path, content string, force bool) error {
	if path == "" || path == "-" {
		return errors.New("--write requires an output file path")
	}
	if !force && fileExists(path) {
		return fmt.Errorf("%s already exists; pass --force to overwrite", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
