package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

// clearCIEnv unsets every variable the flags read, so ambient CI or shell
// settings cannot leak into a case or fail the parse.
func clearCIEnv(t *testing.T) {
	t.Helper()
	prefixes := []string{"DRONE_", "GITHUB_", "CI_", "PLUGIN_", "TELEGRAM_", "INPUT_"}
	bare := map[string]bool{
		"GITHUB": true, "DEBUG": true, "FORMAT": true, "SOCKS5": true,
		"PHOTO": true, "DOCUMENT": true, "STICKER": true, "AUDIO": true,
		"VOICE": true, "LOCATION": true, "VENUE": true, "VIDEO": true,
	}
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		matched := bare[key]
		for _, p := range prefixes {
			matched = matched || strings.HasPrefix(key, p)
		}
		if matched {
			t.Setenv(key, "")
			os.Unsetenv(key)
		}
	}
}

func TestNewAppWoodpeckerEnv(t *testing.T) {
	tests := []struct {
		name         string
		env          map[string]string
		wantStatus   string
		wantStarted  int64
		wantDeployTo string
	}{
		{
			name: "woodpecker failure beats hardcoded drone status",
			env: map[string]string{
				"DRONE_BUILD_STATUS":        "success",
				"CI_PIPELINE_STATUS":        "failure",
				"DRONE_BUILD_STARTED":       "0",
				"CI_PIPELINE_STARTED":       "1700000000",
				"CI_PIPELINE_DEPLOY_TARGET": "production",
			},
			wantStatus:   "failure",
			wantStarted:  1700000000,
			wantDeployTo: "production",
		},
		{
			name: "drone only",
			env: map[string]string{
				"DRONE_BUILD_STATUS":  "failure",
				"DRONE_STAGE_STARTED": "1600000000",
				"DRONE_DEPLOY_TO":     "staging",
			},
			wantStatus:   "failure",
			wantStarted:  1600000000,
			wantDeployTo: "staging",
		},
		{
			name: "drone wins for started and deploy target",
			env: map[string]string{
				"DRONE_STAGE_STARTED":       "1600000000",
				"CI_PIPELINE_STARTED":       "1700000000",
				"DRONE_DEPLOY_TO":           "staging",
				"CI_PIPELINE_DEPLOY_TARGET": "production",
			},
			wantStatus:   "success",
			wantStarted:  1600000000,
			wantDeployTo: "staging",
		},
		{
			name: "empty woodpecker values fall back",
			env: map[string]string{
				"CI_PIPELINE_STATUS":  "",
				"DRONE_BUILD_STATUS":  "failure",
				"CI_PIPELINE_STARTED": "",
			},
			wantStatus:  "failure",
			wantStarted: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearCIEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			app := newApp()
			var status, deployTo string
			var started int64
			app.Action = func(c *cli.Context) error {
				status = c.String("build.status")
				started = c.Int64("build.started")
				deployTo = c.String("deploy.to")
				return nil
			}

			require.NoError(t, app.Run([]string{"woodpecker-telegram"}))
			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantStarted, started)
			assert.Equal(t, tt.wantDeployTo, deployTo)
		})
	}
}

func TestUnsetEmptyEnv(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		set       bool
		wantExist bool
	}{
		{"empty value removed", "", true, false},
		{"whitespace value removed", "   ", true, false},
		{"non-empty value kept", "12345", true, true},
		{"unset stays unset", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_UNSET_EMPTY_ENV"
			os.Unsetenv(key)
			if tt.set {
				t.Setenv(key, tt.value)
			}

			unsetEmptyEnv(key)

			if _, ok := os.LookupEnv(key); ok != tt.wantExist {
				t.Errorf("LookupEnv(%q) exist = %v, want %v", key, ok, tt.wantExist)
			}
			if tt.wantExist {
				if got := os.Getenv(key); got != tt.value {
					t.Errorf("Getenv(%q) = %q, want %q", key, got, tt.value)
				}
			}
		})
	}
}
