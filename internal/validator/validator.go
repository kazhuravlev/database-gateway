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

	"github.com/kazhuravlev/just"
)

var (
	ErrBadQuery         = errors.New("bad query")
	ErrComplicatedQuery = errors.New("complicated query")
	ErrAccessDenied     = errors.New("access denied")
	ErrUnknownTable     = errors.New("unknown table")
	ErrUnknownColumn    = errors.New("unknown column")
)

// IsAllowed will tokenize query, validate schema and check access after all.
func IsAllowed(schema *DbSchema, haveAccess func(Vec) bool, query string) error {
	vectors, err := MakeVectors(query)
	if err != nil {
		return fmt.Errorf("make vectors: %w", err)
	}

	if err := ValidateSchema(vectors, schema); err != nil {
		return fmt.Errorf("validate schema: %w", err)
	}

	if err := ValidateAccess(vectors, haveAccess); err != nil {
		return fmt.Errorf("validate access: %w", err)
	}

	return nil
}

// ValidateSchema will check that request contains only allowed(known) columns.
func ValidateSchema(vectors []Vec, schema *DbSchema) error {
	for _, vec := range vectors {
		tbl, ok := schema.GetTable(vec.Tbl)
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

// ValidateAccess check that all vectors is allowed to run.
func ValidateAccess(vectors []Vec, haveAccess func(Vec) bool) error {
	if !just.SliceAll(vectors, haveAccess) {
		return fmt.Errorf("denied operation: %w", ErrAccessDenied)
	}

	return nil
}
