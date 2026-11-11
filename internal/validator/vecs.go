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
	"slices"
	"strings"

	"github.com/kazhuravlev/database-gateway/internal/config"
	parser2 "github.com/kazhuravlev/database-gateway/internal/parser"
)

type Vec struct {
	Op   config.Op
	Tbl  string
	Cols []string
}

func (v Vec) String() string {
	return fmt.Sprintf("%s:%s(%s)", v.Op, v.Tbl, strings.Join(v.Cols, ","))
}

// MakeVectors create vectors from query.
func MakeVectors(query string) ([]Vec, error) { //nolint:cyclop
	res, err := parser2.Parse(query)

	return convertVectors(res, err)
}

//nolint:cyclop // Enumerates mutation and read requirements for every supported statement.
func convertVectors(res []parser2.Vector, err error) ([]Vec, error) {
	if err != nil {
		if errors.Is(err, parser2.ErrNotImplemented) {
			return nil, errors.Join(err, ErrComplicatedQuery)
		}

		return nil, fmt.Errorf("parse sql: %w", err)
	}

	vectors := make([]Vec, 0, len(res))
	for _, res := range res {
		var vector Vec
		switch expr := res.(type) {
		default:
			return nil, fmt.Errorf("unexpected type (%T): %w", expr, ErrBadQuery)
		case parser2.SelectVec:
			vector = Vec{
				Op:   config.OpSelect,
				Tbl:  expr.Tbl,
				Cols: expr.Columns(),
			}
		case parser2.InsertVec:
			vector = Vec{
				Op:   config.OpInsert,
				Tbl:  expr.Tbl,
				Cols: expr.Columns(),
			}
		case parser2.UpdateVec:
			vector = Vec{
				Op:   config.OpUpdate,
				Tbl:  expr.Tbl,
				Cols: expr.Columns(),
			}
		case parser2.DeleteVec:
			vector = Vec{
				Op:   config.OpDelete,
				Tbl:  expr.Tbl,
				Cols: expr.Columns(),
			}
		}
		vectors = append(vectors, vector)
	}

	// Reading through a mutation must not bypass a SELECT policy. A separate
	// UPDATE vector is already emitted for INSERT ... ON CONFLICT DO UPDATE.
	for _, parsed := range res {
		var read Vec
		read.Op = config.OpSelect
		switch expr := parsed.(type) {
		case parser2.InsertVec:
			read.Tbl, read.Cols = expr.Tbl, slices.Concat(expr.Read, expr.Returning)
		case parser2.UpdateVec:
			read.Tbl, read.Cols = expr.Tbl, slices.Concat(expr.Read, expr.Filter, expr.Returning)
		case parser2.DeleteVec:
			read.Tbl, read.Cols = expr.Tbl, slices.Concat(expr.Filter, expr.Returning)
		}
		if len(read.Cols) != 0 {
			slices.Sort(read.Cols)
			read.Cols = slices.Compact(read.Cols)
			vectors = append(vectors, read)
		}
	}
	for i := range vectors {
		slices.Sort(vectors[i].Cols)
	}

	return vectors, nil
}
