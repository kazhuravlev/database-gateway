package app //nolint:testpackage

import (
	"context"
	"sync"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

const targetPolicy = `
package gateway

default allow_target := false
default allow_vector := false

allow_target if {
	"role:user" in input.subjects
	input.target == "pg-1"
}

allow_target if {
	"user:alice@example.com" in input.subjects
	input.target == "pg-2"
}

allow_vector if {
	allow_target
	input.op == "select"
}
`

func TestServiceGetTargets(t *testing.T) {
	t.Parallel()

	targets := []config.Target{
		{
			ID:            "pg-1",
			Description:   "main",
			Tags:          []string{"prod"},
			Type:          "postgres",
			DefaultSchema: "public",
			Tables:        []config.TargetTable{{Table: "public.clients"}},
		},
		{
			ID:            "pg-2",
			Description:   "analytics",
			Tags:          []string{"analytics"},
			Type:          "postgres",
			DefaultSchema: "public",
			Tables:        []config.TargetTable{{Table: "public.events"}},
		},
	}

	testCases := []struct {
		name    string
		user    structs.User
		wantIDs []config.TargetID
	}{
		{
			name:    "allow by role",
			user:    structs.User{ID: "bob@example.com", Role: config.RoleUser},
			wantIDs: []config.TargetID{"pg-1"},
		},
		{
			name:    "allow by user principal and role",
			user:    structs.User{ID: "alice@example.com", Role: config.RoleUser},
			wantIDs: []config.TargetID{"pg-1", "pg-2"},
		},
		{
			name:    "no matching policy",
			user:    structs.User{ID: "admin@example.com", Role: config.RoleAdmin},
			wantIDs: []config.TargetID{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &Service{
				opts: Options{
					targets:    targets,
					users:      config.UsersProviderOIDC{},
					authorizer: mustAuthorizer(t, targetPolicy),
					storage:    nil,
				},
				connsMu: new(sync.RWMutex),
			}

			got, err := svc.GetTargets(context.Background(), tc.user)
			require.NoError(t, err)

			gotIDs := make([]config.TargetID, 0, len(got))
			for _, server := range got {
				gotIDs = append(gotIDs, server.ID)
			}

			require.Equal(t, tc.wantIDs, gotIDs)
		})
	}
}

func TestServiceGetTargetByID(t *testing.T) {
	t.Parallel()

	target := config.Target{
		ID:            "pg-1",
		Description:   "main",
		Tags:          []string{"prod"},
		Type:          "postgres",
		DefaultSchema: "public",
		Tables:        []config.TargetTable{{Table: "public.clients"}},
	}

	user := structs.User{ID: "alice@example.com", Role: config.RoleUser}

	testCases := []struct {
		name       string
		authorizer string
		targetID   config.TargetID
		wantErrIs  error
		wantServer *structs.Server
	}{
		{
			name:       "allowed target",
			authorizer: targetPolicy,
			targetID:   "pg-1",
			wantServer: &structs.Server{
				ID:          "pg-1",
				Description: "main",
				Tags:        []structs.Tag{{Name: "prod"}},
				Type:        "postgres",
				Tables:      []config.TargetTable{{Table: "public.clients"}},
			},
		},
		{
			name: "target exists but forbidden",
			authorizer: `
package gateway
default allow_target := false
default allow_vector := false
`,
			targetID:  "pg-1",
			wantErrIs: ErrNotFound,
		},
		{
			name:       "target does not exist",
			authorizer: targetPolicy,
			targetID:   "pg-unknown",
			wantErrIs:  ErrNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &Service{
				opts: Options{
					targets:    []config.Target{target},
					users:      config.UsersProviderOIDC{},
					authorizer: mustAuthorizer(t, tc.authorizer),
				},
				connsMu: new(sync.RWMutex),
			}

			got, err := svc.GetTargetByID(context.Background(), user, tc.targetID)
			if tc.wantErrIs != nil {
				require.ErrorIs(t, err, tc.wantErrIs)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantServer, got)
		})
	}
}

func TestPolicyReceivesCanonicalTableName(t *testing.T) {
	t.Parallel()

	schema := validator.NewDbSchema("public", []config.TargetTable{
		{Table: "public.clients", Fields: []string{"id", "name"}},
	})

	var seenTable string
	haveAccess := func(vec validator.Vec) bool {
		seenTable = schema.CanonicalTable(vec.Tbl)

		return seenTable == "public.clients"
	}

	vectors, err := validator.MakeVectors("select id, name from clients")
	require.NoError(t, err)
	require.NoError(t, validator.ValidateSchema(vectors, schema))
	require.NoError(t, validator.ValidateAccess(vectors, haveAccess))
	require.Equal(t, "public.clients", seenTable)
}
