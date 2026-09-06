// Database Gateway provides access to servers with ACL for safe and restricted database interactions.
// Copyright (C) 2024  Kirill Zhuravlev
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package config_test //nolint:exhaustruct

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		prepare         func(cfg *config.Config)
		wantErrContains string
	}{
		{
			name:    "valid config with schema-qualified table",
			prepare: func(_ *config.Config) {},
		},
		{
			name: "already-qualified table",
			prepare: func(cfg *config.Config) {
				cfg.Targets[0].Tables[0].Table = "custom.clients"
			},
		},
		{
			name: "schema-qualified table is required",
			prepare: func(cfg *config.Config) {
				cfg.Targets[0].Tables[0].Table = "clients"
			},
			wantErrContains: "use table notation with leading schema",
		},
		{
			name: "missing policy path",
			prepare: func(cfg *config.Config) {
				cfg.Policy.Path = ""
			},
			wantErrContains: "policy.path is required",
		},
		{
			name: "unsupported role",
			prepare: func(cfg *config.Config) {
				cfg.Users.RoleMapping["broken-group"] = config.Role("owner")
			},
			wantErrContains: `unsupported role "owner" for users.role_mapping["broken-group"]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfigForTest()
			tc.prepare(&cfg)

			err := cfg.Validate()
			if tc.wantErrContains != "" {
				require.ErrorContains(t, err, tc.wantErrContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseFilePolicyPathResolution(t *testing.T) { //nolint:paralleltest // t.Chdir is incompatible with t.Parallel
	cfgDir := t.TempDir()

	// Run from a different working directory to make sure resolution is not cwd-based.
	t.Chdir(t.TempDir())

	writeConfig := func(policyPath string) string {
		t.Helper()

		filename := filepath.Join(cfgDir, "config.json")
		body := `{"policy": {"path": ` + strconv.Quote(policyPath) + `}}`
		require.NoError(t, os.WriteFile(filename, []byte(body), 0o600))

		return filename
	}

	t.Run("relative path is resolved from config file dir", func(t *testing.T) { //nolint:paralleltest
		cfg, err := config.ParseFile(writeConfig("./opa/basic"))
		require.NoError(t, err)
		require.Equal(t, filepath.Join(cfgDir, "opa", "basic"), cfg.Policy.Path)
	})

	t.Run("absolute path is kept as-is", func(t *testing.T) { //nolint:paralleltest
		absPath := filepath.Join(t.TempDir(), "opa", "basic")

		cfg, err := config.ParseFile(writeConfig(absPath))
		require.NoError(t, err)
		require.Equal(t, absPath, cfg.Policy.Path)
	})

	t.Run("relative config filename still resolves to absolute policy path", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(cfgDir)

		cfg, err := config.ParseFile(filepath.Base(writeConfig("./opa/basic")))
		require.NoError(t, err)
		require.Equal(t, filepath.Join(cfgDir, "opa", "basic"), cfg.Policy.Path)
	})
}

func validConfigForTest() config.Config {
	return config.Config{
		Targets: []config.Target{
			{
				ID:          "pg-1",
				Description: "",
				Tags:        nil,
				Type:        "",
				Connection: config.Connection{
					Host:        "",
					Port:        0,
					User:        "",
					Password:    "",
					DB:          "",
					UseSSL:      false,
					MaxPoolSize: 0,
				},
				DefaultSchema: "",
				Tables: []config.TargetTable{
					{Table: "public.known", Fields: nil},
				},
			},
		},
		Users: config.UsersProviderOIDC{
			ClientID:            "",
			ClientSecret:        "",
			IssuerURL:           "",
			RedirectURL:         "",
			Scopes:              nil,
			AccessTokenAudience: "",
			RoleClaim:           "groups",
			RoleMapping: map[string]config.Role{
				"dbgw-admins": config.RoleAdmin,
				"dbgw-users":  config.RoleUser,
			},
		},
		Policy: config.PolicyConfig{
			Path: "./opa",
		},
		Facade: config.FacadeConfig{
			Port:               0,
			CookieSecret:       "",
			UnsafeCORSAllowAll: false,
		},
		Storage: config.PostgresConfig{
			Host:        "",
			Port:        0,
			Database:    "",
			Username:    "",
			Password:    "",
			UseSSL:      false,
			MaxPoolSize: 0,
		},
	}
}
