package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRunConfigTrojanFallbackWritesMachineInstance(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	input := `
instances:
  - id: machine-12
    panel:
      url: "https://panel.example.com"
    machine:
      machine_id: 12
      token_env: "MACHINE_TOKEN"
    kernel:
      type: "singbox"
      config_dir: "/etc/XboardNode-Plus/instances/machine-12"
    log:
      level: "info"
      output: "stdout"
    node:
      access_report_interval: 7
`
	if err := os.WriteFile(cfgPath, []byte(input), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := runConfigTrojanFallback([]string{"--config", cfgPath, "--target", "127.0.0.1:8080"}); err != nil {
		t.Fatalf("runConfigTrojanFallback: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), `trojan_fallback: 127.0.0.1:8080`) {
		t.Fatalf("config missing trojan_fallback:\n%s", string(data))
	}
	if !strings.Contains(string(data), `access_report_interval: 7`) {
		t.Fatalf("config dropped access_report_interval:\n%s", string(data))
	}

	var root fileRootConfig
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("parse written config: %v", err)
	}
	if len(root.Instances) != 1 || root.Instances[0].Node == nil {
		t.Fatalf("written instance node config missing: %#v", root.Instances)
	}
	if got := root.Instances[0].Node.TrojanFallback; got != "127.0.0.1:8080" {
		t.Fatalf("trojan_fallback = %q", got)
	}
	if got := root.Instances[0].Node.AccessReportInterval; got != 7 {
		t.Fatalf("access_report_interval = %d", got)
	}
}
