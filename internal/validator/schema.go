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

package validator

import (
	"strings"

	"github.com/kazhuravlev/database-gateway/internal/config"
)

type DbSchema struct {
	// defaultSchema is a default schema in database. Usually - `public`.
	defaultSchema string
	tables        []config.TargetTable
}

func NewDbSchema(defaultSchema string, tables []config.TargetTable) *DbSchema {
	return &DbSchema{
		defaultSchema: defaultSchema,
		tables:        tables,
	}
}

// GetTable returns table when tblName is registered in DbSchema.tables event with/without schema.
func (s *DbSchema) GetTable(tblName string) (config.TargetTable, bool) {
	tblNamesForLookup := []string{tblName}
	if !strings.Contains(tblName, ".") {
		// Add default schema in case table have no schema
		tblNamesForLookup = append(tblNamesForLookup, s.defaultSchema+"."+tblName)
	}

	for _, t := range s.tables {
		for _, targetTbl := range tblNamesForLookup {
			if t.Table == targetTbl {
				return t, true
			}
		}
	}

	return config.TargetTable{}, false //nolint:exhaustruct
}
