package validator

import (
	"errors"
	"fmt"
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

	vectors, err := makeVectors(query)
	if err != nil {
		return fmt.Errorf("make vectors from query: %w", err)
	}

	if err := validateAccess(vectors, acls); err != nil {
		return fmt.Errorf("validate access: %w", err)
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
