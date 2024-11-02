package validator

import (
	"errors"
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/just"
)

var ErrBadQuery = errors.New("bad query")
var ErrComplicatedQuery = errors.New("complicated query")
var ErrAccessDenied = errors.New("access denied")

type IVector interface {
	Op() config.Op
	String() string
	Table() string
}

func IsAllowed(target config.Target, user config.User, query string) error {
	acls := just.SliceFilter(user.Acls, func(acl config.ACL) bool {
		return acl.Target == target.Id
	})
	if len(acls) == 0 {
		return fmt.Errorf("user have no any acls: %w", ErrAccessDenied)
	}

	stmts, err := parser.Parse(query)
	if err != nil {
		return fmt.Errorf("parse query: %w", err)
	}

	switch len(stmts) {
	default:
		return fmt.Errorf("query contains more than one statement: %w", ErrBadQuery)
	case 0:
		return fmt.Errorf("query contains no statements: %w", ErrBadQuery)
	case 1:
	}

	stmt := stmts[0]

	if tree.CanModifySchema(stmt.AST) {
		return fmt.Errorf("unable to modify schema: %w", ErrBadQuery)
	}

	var crudRequests []tree.NodeFormatter
	collect := func(n tree.NodeFormatter) {
		switch n := n.(type) {
		case *tree.Insert, *tree.Select, *tree.Update, *tree.Delete:
			crudRequests = append(crudRequests, n)
		}
	}
	if err := Walk3(collect, stmt.AST); err != nil {
		return fmt.Errorf("walk statement ast: %w", err)
	}

	if len(crudRequests) == 0 {
		return fmt.Errorf("unsupported query: %w", ErrBadQuery)
	}

	vectors := make([]IVector, 0, len(crudRequests))
	for _, req := range crudRequests {
		switch req := req.(type) {
		default:
			return fmt.Errorf("unexpected query: %w", ErrBadQuery)
		case *tree.Insert:
			vec, err := makeInsertVec(req)
			if err != nil {
				return fmt.Errorf("make insert vec: %w", err)
			}
			vectors = append(vectors, vec)
		case *tree.Select:
			vec, err := MakeSelectVec(req)
			if err != nil {
				return fmt.Errorf("make select vec: %w", err)
			}
			vectors = append(vectors, vec)
		case *tree.Update:
			vec, err := makeUpdateVec(req)
			if err != nil {
				return fmt.Errorf("make update vec: %w", err)
			}
			vectors = append(vectors, vec)
		case *tree.Delete:
			vec, err := makeDeleteVec(req)
			if err != nil {
				return fmt.Errorf("make delete vec: %w", err)
			}
			vectors = append(vectors, vec)
		}
	}

	// Find acl for each vector.
	for _, vec := range vectors {
		isAllowed := false
		for _, acl := range acls {
			if acl.Op != vec.Op() {
				continue
			}

			if acl.Tbl != vec.Table() {
				continue
			}

			isAllowed = acl.Allow
			break
		}

		if !isAllowed {
			return fmt.Errorf("denied operation (%s): %w", vec.String(), ErrAccessDenied)
		}
	}

	return nil
}
