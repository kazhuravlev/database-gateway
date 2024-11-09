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
	"errors"
	"fmt"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/just"
)

var (
	ErrBadQuery         = errors.New("bad query")
	ErrComplicatedQuery = errors.New("complicated query")
	ErrAccessDenied     = errors.New("access denied")
	ErrUnknownTable     = errors.New("unknown table")
	ErrUnknownColumn    = errors.New("unknown column")
)

func IsAllowed(tables []config.TargetTable, haveAccess func(Vec) bool, query string) error {
	vectors, err := makeVectors(query)
	if err != nil {
		return fmt.Errorf("make vectors: %w", err)
	}

	if err := validateSchema(vectors, tables); err != nil {
		return fmt.Errorf("validate schema: %w", err)
	}

	if err := validateAccess(vectors, haveAccess); err != nil {
		return fmt.Errorf("validate access: %w", err)
	}

	return nil
}

// validateSchema will check that request contains only allowed columns.
func validateSchema(vectors []Vec, tables []config.TargetTable) error {
	tblMap := just.Slice2MapFn(tables, func(_ int, tbl config.TargetTable) (string, config.TargetTable) {
		return tbl.Table, tbl
	})
	for _, vec := range vectors {
		tbl, ok := tblMap[vec.Tbl]
		if !ok {
			return fmt.Errorf("not known table: %w", errors.Join(ErrUnknownTable, ErrAccessDenied))
		}

		fMap := just.Slice2Map(tbl.Fields)

		for _, col := range vec.Cols {
			if !just.MapContainsKey(fMap, col) {
				return fmt.Errorf("unable to access column (%s.%s): %w", vec.Tbl, col, errors.Join(ErrUnknownColumn, ErrAccessDenied))
			}
		}
	}

	return nil
}

func validateAccess(vectors []Vec, haveAccess func(Vec) bool) error {
	// Find acl for each vector.
	for _, vec := range vectors {
		if !haveAccess(vec) {
			return fmt.Errorf("denied operation (%s): %w", vec.String(), ErrAccessDenied)
		}
	}

	return nil
}
