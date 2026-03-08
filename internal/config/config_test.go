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

package config_test

import (
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/app/rules"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		prepare func(cfg *config.Config)
		wantErr bool
	}{
		{
			name:    "valid config",
			prepare: func(_ *config.Config) {},
			wantErr: false,
		},
		{
			name: "table without schema prefix",
			prepare: func(cfg *config.Config) {
				cfg.Targets[0].Tables[0].Table = "clients"
			},
			wantErr: true,
		},
		{
			name: "acl references unknown table",
			prepare: func(cfg *config.Config) {
				cfg.ACLs = []rules.ACL{
					{
						User:   "role:user",
						Op:     "select",
						Target: "pg-1",
						Tbl:    "public.missing",
						Allow:  true,
					},
				}
			},
			wantErr: true,
		},
		{
			name: "acl with wildcard table is allowed",
			prepare: func(cfg *config.Config) {
				cfg.ACLs = []rules.ACL{
					{
						User:   "role:user",
						Op:     "select",
						Target: "pg-1",
						Tbl:    rules.Star,
						Allow:  true,
					},
				}
			},
			wantErr: false,
		},
		{
			name: "acl with wildcard target is allowed",
			prepare: func(cfg *config.Config) {
				cfg.ACLs = []rules.ACL{
					{
						User:   "role:user",
						Op:     "select",
						Target: rules.Star,
						Tbl:    "public.known",
						Allow:  true,
					},
				}
			},
			wantErr: false,
		},
		{
			name: "invalid role mapping",
			prepare: func(cfg *config.Config) {
				cfg.Users.RoleMapping["broken-group"] = config.Role("owner")
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfigForTest()
			tc.prepare(&cfg)

			err := cfg.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
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
		ACLs: []rules.ACL{
			{
				User:   "role:user",
				Op:     "select",
				Target: "pg-1",
				Tbl:    "public.known",
				Allow:  true,
			},
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
