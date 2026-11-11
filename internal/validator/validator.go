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
	"github.com/kazhuravlev/database-gateway/internal/parser"
	"github.com/kazhuravlev/just"
)

// AuthorizedQuery can only contain SQL produced and authorized by this package.
// Its zero value is invalid; raw user SQL cannot be substituted after preflight.
type AuthorizedQuery struct {
	sql          string
	vectorsCount int
}

func (q AuthorizedQuery) SQL() string       { return q.sql }
func (q AuthorizedQuery) VectorsCount() int { return q.vectorsCount }

func Authorize(schema *DbSchema, haveAccess func(Vec) bool, query string) (AuthorizedQuery, error) {
	if schema == nil {
		return AuthorizedQuery{}, fmt.Errorf("missing schema: %w", ErrAccessDenied)
	}
	sql, parsed, err := parser.Compile(query, schema.defaultSchema)
	vectors, err := convertVectors(parsed, err)
	if err != nil {
		return AuthorizedQuery{}, fmt.Errorf("make vectors: %w", err)
	}
	if err := ValidateSchema(vectors, schema); err != nil {
		return AuthorizedQuery{}, fmt.Errorf("validate schema: %w", err)
	}
	if err := ValidateAccess(vectors, haveAccess); err != nil {
		return AuthorizedQuery{}, fmt.Errorf("validate access: %w", err)
	}

	return AuthorizedQuery{sql: sql, vectorsCount: len(vectors)}, nil
}

var (
	ErrBadQuery         = errors.New("bad query")
	ErrComplicatedQuery = errors.New("complicated query")
	ErrAccessDenied     = errors.New("access denied")
	ErrUnknownTable     = errors.New("unknown table")
	ErrUnknownColumn    = errors.New("unknown column")
)

// IsAllowed will tokenize query, validate schema and check access after all.
func IsAllowed(schema *DbSchema, haveAccess func(Vec) bool, query string) error {
	_, err := Authorize(schema, haveAccess, query)

	return err
}

// ValidateSchema will check that request contains only allowed(known) columns.
func ValidateSchema(vectors []Vec, schema *DbSchema) error {
	if schema == nil || len(vectors) == 0 {
		return fmt.Errorf("empty schema or requirements: %w", ErrAccessDenied)
	}
	for _, vec := range vectors {
		if !validVector(vec) {
			return fmt.Errorf("invalid requirement: %w", ErrAccessDenied)
		}
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
	if len(vectors) == 0 || haveAccess == nil {
		return fmt.Errorf("missing requirements or authorizer: %w", ErrAccessDenied)
	}
	for _, vec := range vectors {
		if !validVector(vec) {
			return fmt.Errorf("invalid requirement: %w", ErrAccessDenied)
		}
	}
	if !just.SliceAll(vectors, haveAccess) {
		return fmt.Errorf("denied operation: %w", ErrAccessDenied)
	}

	return nil
}

func validVector(vec Vec) bool {
	if vec.Tbl == "" {
		return false
	}
	switch vec.Op {
	case config.OpSelect, config.OpInsert, config.OpUpdate, config.OpDelete:
		return true
	default:
		return false
	}
}
