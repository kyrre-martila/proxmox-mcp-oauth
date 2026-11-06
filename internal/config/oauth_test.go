package config

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestOAuthConfigAndSecretFile(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("test-only-password"), 10)
	path := filepath.Join(t.TempDir(), "admin")
	if e := os.WriteFile(path, append(hash, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
	e := env("PROXMOX_URL", "https://pve:8006", "MCP_TRANSPORT", "http",
		"MCP_OAUTH_ISSUER", "https://mcp.example", "MCP_OAUTH_ADMIN_HASH_FILE", path,
		"MCP_OAUTH_STATE_DIR", "/state", "MCP_OAUTH_REDIRECT_URIS", "https://chatgpt.com/connector_platform_oauth_redirect")
	cfg, err := Load(e)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.OAuth.AdminHash != string(hash) || cfg.HTTP.AuthToken != "" || cfg.Tools.AllowWrites || cfg.Tools.AllowDeletes || cfg.Tools.AllowExec {
		t.Fatal("OAuth changed permissions or secret loading")
	}
	if err = CheckHTTPAuth(cfg.HTTP); err != nil {
		t.Fatal(err)
	}
	cfg.HTTP.AllowUnauthenticated = true
	if CheckHTTPAuth(cfg.HTTP) == nil {
		t.Fatal("unauthenticated mode accepted")
	}
	cfg.HTTP.AllowUnauthenticated = false
	cfg.HTTP.AuthToken = "short"
	if CheckHTTPAuth(cfg.HTTP) == nil {
		t.Fatal("weak static token accepted")
	}
	cfg.HTTP.AuthToken = "valid-static-token"
	if CheckHTTPAuth(cfg.HTTP) != nil {
		t.Fatal("legacy auth rejected")
	}
}
