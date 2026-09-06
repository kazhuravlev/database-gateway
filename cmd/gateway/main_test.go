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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeRunConfig writes a minimal config for the "run" command. Storage points at an
// unreachable port so any accidental DB access fails fast and visibly.
func writeRunConfig(t *testing.T, policyDir string) string {
	t.Helper()

	cfg := fmt.Sprintf(`{
		"targets": [],
		"users": {
			"client_id": "x",
			"client_secret": "x",
			"issuer_url": "http://127.0.0.1:1/",
			"redirect_url": "http://127.0.0.1:1/cb",
			"scopes": [],
			"access_token_audience": "x",
			"role_claim": "groups",
			"role_mapping": {"g": "user"}
		},
		"policy": {"path": %q},
		"facade": {"port": 0, "cookie_secret": "x", "unsafe_cors_allow_all": false},
		"storage": {
			"host": "127.0.0.1",
			"port": 1,
			"database": "x",
			"username": "x",
			"password": "x",
			"use_ssl": false,
			"max_pool_size": 1
		}
	}`, policyDir)

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfg), 0o600))

	return cfgPath
}

// The service must refuse to start when any .rego module fails to compile:
// the run action returns an error naming the broken file, which makes main()
// exit with a non-zero code.
func TestRunFailsOnInvalidPolicy(t *testing.T) {
	t.Parallel()

	policyDir := t.TempDir()
	brokenFile := filepath.Join(policyDir, "broken.rego")
	require.NoError(t, os.WriteFile(brokenFile, []byte("package gateway\n\nallow_target :=\n"), 0o600))

	err := newCLIApp().Run([]string{"gateway", "-c", writeRunConfig(t, policyDir), "run"})
	require.Error(t, err)
	require.ErrorContains(t, err, "load policy modules")
	require.ErrorContains(t, err, brokenFile)
	require.ErrorContains(t, err, "rego_parse_error")
}

// With compilable policies the startup must get past policy loading; here it
// fails later, on the (intentionally unreachable) metadata database. This pins
// the fail-fast ordering: policies are checked before any DB access.
func TestRunWithValidPolicyReachesDatabase(t *testing.T) {
	t.Parallel()

	policyDir := t.TempDir()
	validPolicy := "package gateway\n\ndefault allow_target := false\ndefault allow_query := false\n"
	require.NoError(t, os.WriteFile(filepath.Join(policyDir, "policy.rego"), []byte(validPolicy), 0o600))

	err := newCLIApp().Run([]string{"gateway", "-c", writeRunConfig(t, policyDir), "run"})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "load policy modules")
	require.ErrorContains(t, err, "create new migrator")
}
