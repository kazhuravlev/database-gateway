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

package pgdb_test

import (
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/pgdb"
	"github.com/stretchr/testify/assert"
)

func TestBuildDBDsn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.PostgresConfig
		want string
	}{
		{
			name: "ssl disabled",
			cfg: config.PostgresConfig{
				Host:        "localhost",
				Port:        5432,
				Username:    "gateway",
				Password:    "secret",
				Database:    "metadata",
				UseSSL:      false,
				MaxPoolSize: 5,
			},
			want: "host=localhost port=5432 user=gateway password=secret dbname=metadata sslmode=disable",
		},
		{
			name: "ssl enabled",
			cfg: config.PostgresConfig{
				Host:        "db.example.com",
				Port:        6432,
				Username:    "readonly",
				Password:    "password",
				Database:    "gateway",
				UseSSL:      true,
				MaxPoolSize: 10,
			},
			want: "host=db.example.com port=6432 user=readonly password=password dbname=gateway sslmode=prefer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, pgdb.BuildDBDsn(tt.cfg))
		})
	}
}
