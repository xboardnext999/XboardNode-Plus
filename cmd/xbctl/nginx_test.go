package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderNginxTrojanSNIFull(t *testing.T) {
	got, err := renderNginxTrojanSNI(nginxTrojanSNIOptions{
		Domain:         "trojan.example.com",
		WebDomain:      "www.example.com",
		TrojanUpstream: "127.0.0.1:10443",
		WebUpstream:    "127.0.0.1:8443",
		Listens:        []string{"443", "[::]:443"},
		DefaultBackend: "web",
		Format:         "full",
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"stream {",
		"map $ssl_preread_server_name $xboard_node_backend",
		"trojan.example.com xboard_node_trojan;",
		"www.example.com xboard_node_web;",
		"default xboard_node_web;",
		"server 127.0.0.1:10443;",
		"server 127.0.0.1:8443;",
		"listen 443 reuseport;",
		"listen [::]:443 reuseport;",
		"ssl_preread on;",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered config missing %q:\n%s", want, got)
		}
	}
}

func TestRenderNginxTrojanSNIRejectsUnsafeToken(t *testing.T) {
	_, err := renderNginxTrojanSNI(nginxTrojanSNIOptions{
		Domain:         "trojan.example.com;include",
		TrojanUpstream: "127.0.0.1:10443",
		WebUpstream:    "127.0.0.1:8443",
		Listens:        []string{"443"},
		DefaultBackend: "web",
		Format:         "snippet",
	})
	if err == nil {
		t.Fatal("expected unsafe token error")
	}
}

func TestRunNginxTrojanSNIWriteDefaultsToSnippet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trojan.conf")
	err := runNginxTrojanSNI([]string{
		"--domain", "trojan.example.com",
		"--output", path,
		"--write",
	})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if strings.HasPrefix(strings.TrimSpace(got), "stream {") {
		t.Fatalf("write mode should default to snippet without outer stream block:\n%s", got)
	}
	if !strings.Contains(got, "map $ssl_preread_server_name $xboard_node_backend") {
		t.Fatalf("rendered snippet missing map:\n%s", got)
	}
}
