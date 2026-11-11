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

package validator_test

import (
	"fmt"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

func restrictedSchema() *validator.DbSchema {
	return validator.NewDbSchema("public", []config.TargetTable{{Table: "public.clients", Fields: []string{"id", "name"}}})
}

func TestAuthorizationRejectsHiddenReads(t *testing.T) {
	t.Parallel()
	for _, query := range []string{
		"SELECT secret AS id FROM clients",
		"DELETE FROM clients RETURNING secret AS id",
		"UPDATE clients SET id=1 RETURNING secret AS id",
		"INSERT INTO clients(id) VALUES(1) RETURNING secret AS id",
		"INSERT INTO clients(id) VALUES(1) ON CONFLICT(id) DO UPDATE SET id=secret",
		"INSERT INTO clients(id) VALUES(1) ON CONFLICT(id) DO UPDATE SET id=clients.secret+1",
		"SELECT c.id FROM clients c JOIN secrets clients ON c.id=clients.id",
		"SELECT secrets.id FROM clients secrets JOIN secrets clients ON secrets.id=clients.id",
	} {
		t.Run(query, func(t *testing.T) {
			t.Parallel()
			calls := 0
			authorized, err := validator.Authorize(restrictedSchema(), func(validator.Vec) bool {
				calls++

				return true
			}, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
			require.Empty(t, authorized.SQL())
			require.Zero(t, authorized.VectorsCount())
			require.Zero(t, calls, "schema rejection must precede policy evaluation")
		})
	}
}

func TestEveryOperationMustBeAuthorized(t *testing.T) {
	t.Parallel()
	cases := []struct {
		query      string
		operations []config.Op
	}{
		{"INSERT INTO clients(id) VALUES(1) ON CONFLICT(id) DO UPDATE SET id=2 RETURNING id", []config.Op{config.OpInsert, config.OpUpdate, config.OpSelect}},
		{"UPDATE clients SET id=id+1", []config.Op{config.OpUpdate, config.OpSelect}},
		{"UPDATE clients SET name='x' WHERE id=1", []config.Op{config.OpUpdate, config.OpSelect}},
		{"DELETE FROM clients RETURNING id", []config.Op{config.OpDelete, config.OpSelect}},
		{"INSERT INTO clients(id) VALUES(1) RETURNING id", []config.Op{config.OpInsert, config.OpSelect}},
	}
	for _, tc := range cases {
		for _, denied := range tc.operations {
			t.Run(tc.query+"/deny_"+denied.S(), func(t *testing.T) {
				t.Parallel()
				authorized, err := validator.Authorize(restrictedSchema(), func(v validator.Vec) bool { return v.Op != denied }, tc.query)
				require.ErrorIs(t, err, validator.ErrAccessDenied)
				require.Empty(t, authorized.SQL())
			})
		}
		authorized, err := validator.Authorize(restrictedSchema(), func(validator.Vec) bool { return true }, tc.query)
		require.NoError(t, err)
		require.NotEmpty(t, authorized.SQL())
	}
}

func TestAuthorizationNeverAcceptsEmptyRequirements(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, validator.ValidateSchema(nil, nil), validator.ErrAccessDenied)
	require.ErrorIs(t, validator.ValidateSchema(nil, restrictedSchema()), validator.ErrAccessDenied)
	require.ErrorIs(t, validator.ValidateAccess(nil, func(validator.Vec) bool { return true }), validator.ErrAccessDenied)
	require.ErrorIs(t, validator.ValidateAccess([]validator.Vec{{Op: config.OpSelect, Tbl: "public.clients"}}, nil), validator.ErrAccessDenied)
	require.ErrorIs(t, validator.ValidateAccess([]validator.Vec{{Op: "execute", Tbl: "public.clients"}}, func(validator.Vec) bool { return true }), validator.ErrAccessDenied)
}

// Generate supported SQL with a forbidden dependency in each expression slot.
// Renaming and numeric literal mutations must never erase that dependency.
func FuzzAuthorizationCannotDropColumns(f *testing.F) {
	f.Add(uint8(0), int32(1))
	f.Add(uint8(5), int32(-1))
	f.Fuzz(func(t *testing.T, slot uint8, value int32) {
		templates := []string{
			"SELECT secret FROM clients WHERE id=%d",
			"SELECT id FROM clients WHERE secret=%d",
			"SELECT id FROM clients WHERE id=%d ORDER BY secret",
			"SELECT id FROM clients WHERE id=%d GROUP BY id,secret",
			"UPDATE clients SET id=%d RETURNING secret AS name",
			"INSERT INTO clients(id) VALUES(%d) ON CONFLICT(id) DO UPDATE SET id=secret",
			"DELETE FROM clients WHERE id=%d RETURNING secret AS id",
		}
		query := fmt.Sprintf(templates[int(slot)%len(templates)], value)
		authorized, err := validator.Authorize(restrictedSchema(), func(validator.Vec) bool { return true }, query)
		require.Error(t, err)
		require.Empty(t, authorized.SQL())
	})
}
