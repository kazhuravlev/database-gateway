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

type IVector interface {
	Op() config.Op
	String() string
	Table() string
	Columns() []string
}

func IsAllowed(target config.Target, user config.User, query string) error {
	acls := just.SliceFilter(user.Acls, func(acl config.ACL) bool {
		return acl.Target == target.Id
	})
	if len(acls) == 0 {
		return fmt.Errorf("user have no any acls: %w", ErrAccessDenied)
	}

	vectors, err := makeVectors(query)
	if err != nil {
		return fmt.Errorf("make vectors from query: %w", err)
	}

	if err := validateSchema(vectors, target.Tables); err != nil {
		return fmt.Errorf("validate schema: %w", err)
	}

	if err := validateAccess(vectors, acls); err != nil {
		return fmt.Errorf("validate access: %w", err)
	}

	return nil
}

// validateSchema will check that request contains only allowed columns.
func validateSchema(vectors []IVector, tables []config.TargetTable) error {
	tblMap := just.Slice2MapFn(tables, func(_ int, tbl config.TargetTable) (string, config.TargetTable) {
		return tbl.Table, tbl
	})
	for _, vec := range vectors {
		tbl, ok := tblMap[vec.Table()]
		if !ok {
			return fmt.Errorf("not known table: %w", errors.Join(ErrUnknownTable, ErrAccessDenied))
		}

		fMap := just.Slice2Map(tbl.Fields)

		for _, col := range vec.Columns() {
			if !just.MapContainsKey(fMap, col) {
				return fmt.Errorf("unable to access column (%s.%s): %w", vec.Table(), col, errors.Join(ErrUnknownColumn, ErrAccessDenied))
			}
		}
	}

	return nil
}

func validateAccess(vectors []IVector, acls []config.ACL) error {
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
