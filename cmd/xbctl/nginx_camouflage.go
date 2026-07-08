package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"html"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultNginxCamouflagePath = "/etc/nginx/conf.d/xboard-node-camouflage.conf"
	defaultCamouflageRoot      = "/var/www/xboard-node-camouflage"
)

type nginxCamouflageOptions struct {
	Domain   string
	Listen   string
	Root     string
	Template string
	Output   string
	Write    bool
	Force    bool
	Test     bool
	Reload   bool
}

type camouflageTemplate struct {
	Name        string
	Title       string
	Subtitle    string
	Kicker      string
	Primary     string
	Secondary   string
	Accent      string
	AccentSoft  string
	Metrics     [][2]string
	Features    []string
	Button      string
	FooterLabel string
}

var camouflageTemplates = []camouflageTemplate{
	{
		Name:        "cloud",
		Title:       "Edge Cloud Platform",
		Subtitle:    "Deploy, observe, and protect modern services across a global edge footprint.",
		Kicker:      "Global infrastructure",
		Primary:     "#635bff",
		Secondary:   "#17b26a",
		Accent:      "#0f172a",
		AccentSoft:  "#eef2ff",
		Metrics:     [][2]string{{"99.98%", "service uptime"}, {"42 ms", "median response"}, {"18", "edge regions"}},
		Features:    []string{"Edge acceleration", "Real-time telemetry", "Zero downtime routing"},
		Button:      "View status",
		FooterLabel: "Operational dashboard",
	},
	{
		Name:        "studio",
		Title:       "Northstar Creative Studio",
		Subtitle:    "A digital product team crafting elegant websites, brand systems, and launch pages.",
		Kicker:      "Design and engineering",
		Primary:     "#e8558e",
		Secondary:   "#f59e0b",
		Accent:      "#111827",
		AccentSoft:  "#fff1f2",
		Metrics:     [][2]string{{"128", "projects shipped"}, {"14", "brand launches"}, {"4.9", "client rating"}},
		Features:    []string{"Brand strategy", "Web experience", "Product interface"},
		Button:      "Explore work",
		FooterLabel: "Selected case studies",
	},
	{
		Name:        "docs",
		Title:       "Atlas Knowledge Base",
		Subtitle:    "Guides, changelogs, and operational notes for distributed product teams.",
		Kicker:      "Documentation portal",
		Primary:     "#2563eb",
		Secondary:   "#06b6d4",
		Accent:      "#0f172a",
		AccentSoft:  "#eff6ff",
		Metrics:     [][2]string{{"2.4k", "articles"}, {"86", "collections"}, {"24/7", "search"}},
		Features:    []string{"API guides", "Release notes", "Team playbooks"},
		Button:      "Open docs",
		FooterLabel: "Updated today",
	},
	{
		Name:        "commerce",
		Title:       "Harbor Commerce",
		Subtitle:    "A calm storefront toolkit for subscriptions, invoices, and customer operations.",
		Kicker:      "Commerce suite",
		Primary:     "#0ea5e9",
		Secondary:   "#22c55e",
		Accent:      "#111827",
		AccentSoft:  "#ecfeff",
		Metrics:     [][2]string{{"36", "payment regions"}, {"0.8s", "checkout load"}, {"12k", "orders synced"}},
		Features:    []string{"Checkout pages", "Invoice center", "Member accounts"},
		Button:      "Start checkout",
		FooterLabel: "Commerce workspace",
	},
	{
		Name:        "status",
		Title:       "Pulse Service Status",
		Subtitle:    "Live health checks and incident updates for customer-facing services.",
		Kicker:      "System status",
		Primary:     "#16a34a",
		Secondary:   "#84cc16",
		Accent:      "#0f172a",
		AccentSoft:  "#f0fdf4",
		Metrics:     [][2]string{{"All", "systems normal"}, {"0", "open incidents"}, {"31", "monitors"}},
		Features:    []string{"Availability checks", "Incident timeline", "Subscriber alerts"},
		Button:      "Subscribe",
		FooterLabel: "No active incidents",
	},
}

func runNginxCamouflageSite(args []string) error {
	opts := nginxCamouflageOptions{
		Listen:   "127.0.0.1:8080",
		Root:     defaultCamouflageRoot,
		Template: "random",
		Output:   "-",
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--domain":
			if i+1 >= len(args) {
				return errors.New("missing value for --domain")
			}
			i++
			opts.Domain = args[i]
		case "--listen":
			if i+1 >= len(args) {
				return errors.New("missing value for --listen")
			}
			i++
			opts.Listen = args[i]
		case "--root":
			if i+1 >= len(args) {
				return errors.New("missing value for --root")
			}
			i++
			opts.Root = args[i]
		case "--template":
			if i+1 >= len(args) {
				return errors.New("missing value for --template")
			}
			i++
			opts.Template = args[i]
		case "--output", "-o":
			if i+1 >= len(args) {
				return errors.New("missing value for --output")
			}
			i++
			opts.Output = args[i]
		case "--write":
			opts.Write = true
			if opts.Output == "-" {
				opts.Output = defaultNginxCamouflagePath
			}
		case "--force":
			opts.Force = true
		case "--test":
			opts.Test = true
		case "--reload":
			opts.Reload = true
		case "--help", "-h":
			printNginxCamouflageUsage()
			return nil
		default:
			return fmt.Errorf("unknown camouflage-site arg: %s", args[i])
		}
	}

	tpl, err := selectCamouflageTemplate(opts.Template)
	if err != nil {
		return err
	}
	conf, err := renderNginxCamouflageSite(opts)
	if err != nil {
		return err
	}
	htmlDoc := renderCamouflageHTML(tpl, opts.Domain)

	if opts.Write {
		if err := writeCamouflageSite(opts.Root, htmlDoc, opts.Force); err != nil {
			return err
		}
		if err := writeNginxConfig(opts.Output, conf, opts.Force); err != nil {
			return err
		}
		fmt.Printf("Wrote %s\n", filepath.Join(opts.Root, "index.html"))
		fmt.Printf("Wrote %s\n", opts.Output)
		fmt.Printf("Template: %s\n", tpl.Name)
	} else {
		fmt.Printf("# Template: %s\n", tpl.Name)
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

func printNginxCamouflageUsage() {
	fmt.Println(`xbctl nginx camouflage-site [flags]

Generate a local HTTP camouflage website and nginx server block for Trojan
fallback traffic. This is intended for same-domain Trojan camouflage where the
Trojan inbound terminates TLS and forwards invalid/browser requests to HTTP.

Required:
  --domain DOMAIN              public domain shown by the camouflage site

Optional:
  --listen ADDR:PORT           local HTTP listen address (default 127.0.0.1:8080)
  --root PATH                  static site root (default /var/www/xboard-node-camouflage)
  --template NAME              random|cloud|studio|docs|commerce|status (default random)
  --output PATH, -o PATH       nginx config path when --write is used
  --write                      write site and /etc/nginx/conf.d/xboard-node-camouflage.conf
  --force                      overwrite existing output
  --test                       run nginx -t after output/write
  --reload                     run nginx -s reload after output/write`)
}

func selectCamouflageTemplate(name string) (camouflageTemplate, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" || name == "random" {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(camouflageTemplates))))
		if err != nil {
			return camouflageTemplates[0], nil
		}
		return camouflageTemplates[n.Int64()], nil
	}
	for _, tpl := range camouflageTemplates {
		if tpl.Name == name {
			return tpl, nil
		}
	}
	names := make([]string, 0, len(camouflageTemplates)+1)
	names = append(names, "random")
	for _, tpl := range camouflageTemplates {
		names = append(names, tpl.Name)
	}
	return camouflageTemplate{}, fmt.Errorf("--template must be one of: %s", strings.Join(names, ", "))
}

func renderNginxCamouflageSite(opts nginxCamouflageOptions) (string, error) {
	opts.Domain = strings.TrimSpace(opts.Domain)
	if opts.Domain == "" {
		return "", errors.New("--domain is required")
	}
	if err := validateNginxToken("domain", opts.Domain); err != nil {
		return "", err
	}
	if err := validateNginxToken("listen", opts.Listen); err != nil {
		return "", err
	}
	if err := validateNginxPathToken("root", opts.Root); err != nil {
		return "", err
	}

	var body strings.Builder
	body.WriteString("# Generated by xbctl nginx camouflage-site.\n")
	body.WriteString("# This HTTP server is for Trojan fallback after TLS has been terminated by the node.\n\n")
	body.WriteString("server {\n")
	fmt.Fprintf(&body, "    listen %s;\n", opts.Listen)
	fmt.Fprintf(&body, "    server_name %s;\n", opts.Domain)
	fmt.Fprintf(&body, "    root %s;\n", opts.Root)
	body.WriteString("    index index.html;\n")
	body.WriteString("    access_log off;\n")
	body.WriteString("    add_header X-Content-Type-Options nosniff always;\n")
	body.WriteString("    add_header Referrer-Policy strict-origin-when-cross-origin always;\n")
	body.WriteString("    location / {\n")
	body.WriteString("        try_files $uri $uri/ /index.html;\n")
	body.WriteString("    }\n")
	body.WriteString("}\n")
	return body.String(), nil
}

func validateNginxPathToken(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s cannot be empty", field)
	}
	if strings.ContainsAny(value, "{};\r\n\t ") {
		return fmt.Errorf("%s contains characters that are unsafe for nginx config: %q", field, value)
	}
	return nil
}

func writeCamouflageSite(root, htmlDoc string, force bool) error {
	if root == "" {
		return errors.New("--root cannot be empty")
	}
	indexPath := filepath.Join(root, "index.html")
	if !force && fileExists(indexPath) {
		return fmt.Errorf("%s already exists; pass --force to overwrite", indexPath)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(indexPath, []byte(htmlDoc), 0o644)
}

func renderCamouflageHTML(tpl camouflageTemplate, domain string) string {
	escapedDomain := html.EscapeString(domain)
	metricCards := make([]string, 0, len(tpl.Metrics))
	for _, metric := range tpl.Metrics {
		metricCards = append(metricCards, fmt.Sprintf(`<div class="metric"><strong>%s</strong><span>%s</span></div>`, html.EscapeString(metric[0]), html.EscapeString(metric[1])))
	}
	featureCards := make([]string, 0, len(tpl.Features))
	for _, feature := range tpl.Features {
		featureCards = append(featureCards, fmt.Sprintf(`<li><span></span>%s</li>`, html.EscapeString(feature)))
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <style>
    :root {
      --primary: %s;
      --secondary: %s;
      --accent: %s;
      --soft: %s;
      --muted: #64748b;
      --line: #e5e7eb;
      --page: #f8fafc;
      --card: rgba(255, 255, 255, .86);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--accent);
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background:
        radial-gradient(circle at 12%% 10%%, var(--soft) 0, transparent 28rem),
        radial-gradient(circle at 86%% 16%%, rgba(99, 91, 255, .13) 0, transparent 24rem),
        linear-gradient(135deg, #ffffff 0%%, var(--page) 100%%);
    }
    a { color: inherit; text-decoration: none; }
    .shell { width: min(1120px, calc(100%% - 40px)); margin: 0 auto; padding: 36px 0; }
    .nav, .hero, .panel, .status { border: 1px solid rgba(15, 23, 42, .08); background: var(--card); box-shadow: 0 24px 70px rgba(15, 23, 42, .08); backdrop-filter: blur(18px); }
    .nav { height: 64px; display: flex; align-items: center; justify-content: space-between; padding: 0 20px; border-radius: 24px; }
    .brand { display: flex; align-items: center; gap: 12px; font-weight: 750; letter-spacing: .01em; }
    .mark { width: 34px; height: 34px; border-radius: 12px; background: linear-gradient(135deg, var(--primary), var(--secondary)); box-shadow: inset 0 0 0 1px rgba(255,255,255,.35); }
    .pill { padding: 9px 14px; border: 1px solid var(--line); border-radius: 999px; color: var(--muted); font-size: 14px; background: rgba(255,255,255,.7); }
    .hero { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(280px, .9fr); gap: 26px; margin-top: 24px; padding: 44px; border-radius: 32px; overflow: hidden; }
    .kicker { color: var(--primary); font-weight: 750; font-size: 14px; text-transform: uppercase; letter-spacing: .08em; }
    h1 { margin: 16px 0 16px; max-width: 650px; font-size: clamp(42px, 7vw, 78px); line-height: .95; letter-spacing: -.04em; }
    .subtitle { margin: 0; max-width: 600px; color: var(--muted); font-size: 18px; line-height: 1.7; }
    .actions { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 30px; }
    .button { display: inline-flex; align-items: center; justify-content: center; min-height: 46px; padding: 0 18px; border-radius: 15px; font-weight: 750; background: var(--accent); color: #fff; }
    .button.secondary { color: var(--accent); background: #fff; border: 1px solid var(--line); }
    .panel { border-radius: 28px; padding: 24px; align-self: stretch; }
    .grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 18px; }
    .metric { min-height: 112px; border-radius: 22px; padding: 18px; background: linear-gradient(160deg, #fff, var(--soft)); border: 1px solid rgba(15, 23, 42, .07); }
    .metric strong { display: block; font-size: 30px; letter-spacing: -.03em; }
    .metric span { display: block; margin-top: 6px; color: var(--muted); font-size: 13px; }
    .features { display: grid; gap: 12px; margin: 0; padding: 0; list-style: none; }
    .features li { display: flex; align-items: center; gap: 10px; min-height: 48px; padding: 0 14px; border-radius: 16px; color: #334155; background: rgba(255,255,255,.72); border: 1px solid rgba(15, 23, 42, .06); }
    .features span { width: 9px; height: 9px; border-radius: 50%%; background: var(--primary); box-shadow: 0 0 0 5px var(--soft); }
    .status { display: grid; grid-template-columns: 1fr auto; align-items: center; gap: 18px; margin-top: 24px; padding: 22px 24px; border-radius: 24px; color: var(--muted); }
    .dot { width: 11px; height: 11px; border-radius: 50%%; background: var(--secondary); box-shadow: 0 0 0 7px color-mix(in srgb, var(--secondary) 14%%, transparent); }
    @media (max-width: 760px) {
      .shell { width: min(100%% - 24px, 1120px); padding: 16px 0; }
      .hero { grid-template-columns: 1fr; padding: 28px; }
      .grid { grid-template-columns: 1fr; }
      h1 { font-size: 44px; }
    }
  </style>
</head>
<body>
  <main class="shell">
    <nav class="nav">
      <div class="brand"><span class="mark"></span><span>%s</span></div>
      <div class="pill">%s</div>
    </nav>
    <section class="hero">
      <div>
        <div class="kicker">%s</div>
        <h1>%s</h1>
        <p class="subtitle">%s</p>
        <div class="actions">
          <a class="button" href="/">%s</a>
          <a class="button secondary" href="mailto:hello@%s">Contact</a>
        </div>
      </div>
      <aside class="panel">
        <div class="grid">%s</div>
        <ul class="features">%s</ul>
      </aside>
    </section>
    <section class="status">
      <div><strong>%s</strong><br><span>%s</span></div>
      <span class="dot"></span>
    </section>
  </main>
</body>
</html>
`, html.EscapeString(tpl.Title), tpl.Primary, tpl.Secondary, tpl.Accent, tpl.AccentSoft, escapedDomain, html.EscapeString(tpl.Kicker), html.EscapeString(tpl.Kicker), html.EscapeString(tpl.Title), html.EscapeString(tpl.Subtitle), html.EscapeString(tpl.Button), escapedDomain, strings.Join(metricCards, ""), strings.Join(featureCards, ""), html.EscapeString(tpl.FooterLabel), escapedDomain)
}
