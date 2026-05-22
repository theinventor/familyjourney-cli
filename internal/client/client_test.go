package client

import (
	"testing"

	"github.com/theinventor/familyjourney-cli/internal/config"
	"github.com/theinventor/familyjourney-cli/internal/credstore"
)

func TestNewWithProfileAndEnvResolution(t *testing.T) {
	t.Setenv(config.EnvConfig, t.TempDir()+"/config.json")
	t.Setenv(credstore.EnvDisableKeychain, "1")

	f := &config.File{DefaultProfile: "saved", Profiles: map[string]config.Profile{}}
	f.Put("saved", config.Profile{
		APIURL:   "https://saved.example",
		APIToken: "fj_saved_1234567890",
		Backend:  credstore.BackendFile,
	})
	if err := f.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	saved := New()
	if saved.Source != "profile:saved" || saved.BaseURL != "https://saved.example" || saved.APIToken != "fj_saved_1234567890" {
		t.Fatalf("saved profile resolution mismatch: %#v", saved)
	}

	t.Setenv(EnvAPIToken, "fj_env_1234567890")
	t.Setenv(EnvAPIURL, "https://env.example/")
	env := New()
	if env.Source != "env" || env.BaseURL != "https://env.example" || env.APIToken != "fj_env_1234567890" {
		t.Fatalf("env resolution mismatch: %#v", env)
	}

	explicit := NewWithProfile("saved")
	if explicit.Source != "profile:saved" || explicit.BaseURL != "https://saved.example" || explicit.APIToken != "fj_saved_1234567890" {
		t.Fatalf("explicit profile should win over env: %#v", explicit)
	}
}

func TestMaskToken(t *testing.T) {
	if got := MaskToken(""); got != "(none)" {
		t.Fatalf("empty mask mismatch: %q", got)
	}
	if got := MaskToken("short"); got != "***" {
		t.Fatalf("short mask mismatch: %q", got)
	}
	if got := MaskToken("fj_secret_1234567890"); got != "fj_secre...7890" {
		t.Fatalf("long mask mismatch: %q", got)
	}
}
