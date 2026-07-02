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

package facade //nolint:exhaustruct,testpackage

import (
	"context"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/app"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/lrpc/ctypes"
	"github.com/stretchr/testify/require"
)

const lrpcTestUUID = "01HZ7N9R8Y9V1B9KZ6ZQVV8A1S"

const lrpcTestCallID = ctypes.ID("")

func TestLRPCHandlersRequireAPITokenUserContext(t *testing.T) {
	t.Parallel()

	svc := new(Service)
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "target list",
			call: func() error {
				_, err := svc.lrpcTargetList(ctx, lrpcTestCallID, nil)

				return err
			},
		},
		{
			name: "target get",
			call: func() error {
				_, err := svc.lrpcTargetGet(ctx, lrpcTestCallID, lrpcTargetGetReq{TargetID: "pg-1"})

				return err
			},
		},
		{
			name: "bookmark add",
			call: func() error {
				_, err := svc.lrpcBookmarksAdd(ctx, lrpcTestCallID, lrpcBookmarksAddReq{
					TargetID: "pg-1",
					Title:    "title",
					Query:    "select 1",
				})

				return err
			},
		},
		{
			name: "bookmark delete",
			call: func() error {
				_, err := svc.lrpcBookmarksDelete(ctx, lrpcTestCallID, lrpcBookmarksDeleteReq{ID: lrpcTestUUID})

				return err
			},
		},
		{
			name: "query run",
			call: func() error {
				_, err := svc.lrpcQueryRun(ctx, lrpcTestCallID, lrpcQueryRunReq{
					TargetID: "pg-1",
					Query:    "select 1",
				})

				return err
			},
		},
		{
			name: "query results get",
			call: func() error {
				_, err := svc.lrpcQueryResultsGet(ctx, lrpcTestCallID, lrpcQueryResultsGetReq{
					QueryResultID: lrpcTestUUID,
				})

				return err
			},
		},
		{
			name: "export link",
			call: func() error {
				_, err := svc.lrpcQueryResultsExportLink(ctx, lrpcTestCallID, lrpcQueryResultsExportLinkReq{
					QueryResultID: lrpcTestUUID,
					Format:        "json",
				})

				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.call()
			require.ErrorIs(t, err, app.ErrForbidden)
		})
	}
}

func TestLRPCHandlersRejectBadUUIDInputs(t *testing.T) {
	t.Parallel()

	svc := new(Service)
	ctx := lrpcTestContext()

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "bookmark delete id",
			call: func() error {
				_, err := svc.lrpcBookmarksDelete(ctx, lrpcTestCallID, lrpcBookmarksDeleteReq{ID: "bad-id"})

				return err
			},
		},
		{
			name: "query results get id",
			call: func() error {
				_, err := svc.lrpcQueryResultsGet(ctx, lrpcTestCallID, lrpcQueryResultsGetReq{ID: "bad-id"})

				return err
			},
		},
		{
			name: "query results get query_result_id",
			call: func() error {
				_, err := svc.lrpcQueryResultsGet(ctx, lrpcTestCallID, lrpcQueryResultsGetReq{QueryResultID: "bad-id"})

				return err
			},
		},
		{
			name: "export link query_result_id",
			call: func() error {
				_, err := svc.lrpcQueryResultsExportLink(ctx, lrpcTestCallID, lrpcQueryResultsExportLinkReq{
					QueryResultID: "bad-id",
					Format:        "json",
				})

				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.call()
			require.ErrorIs(t, err, errBadInput)
		})
	}
}

func TestLRPCHandlersRejectTrimmedRequiredFields(t *testing.T) {
	t.Parallel()

	svc := new(Service)
	ctx := lrpcTestContext()

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "target get target_id",
			call: func() error {
				_, err := svc.lrpcTargetGet(ctx, lrpcTestCallID, lrpcTargetGetReq{TargetID: " \t\n "})

				return err
			},
		},
		{
			name: "bookmark add target_id",
			call: func() error {
				_, err := svc.lrpcBookmarksAdd(ctx, lrpcTestCallID, lrpcBookmarksAddReq{
					TargetID: " \t\n ",
					Title:    "title",
					Query:    "select 1",
				})

				return err
			},
		},
		{
			name: "bookmark add title",
			call: func() error {
				_, err := svc.lrpcBookmarksAdd(ctx, lrpcTestCallID, lrpcBookmarksAddReq{
					TargetID: "pg-1",
					Title:    " \t\n ",
					Query:    "select 1",
				})

				return err
			},
		},
		{
			name: "bookmark add query",
			call: func() error {
				_, err := svc.lrpcBookmarksAdd(ctx, lrpcTestCallID, lrpcBookmarksAddReq{
					TargetID: "pg-1",
					Title:    "title",
					Query:    " \t\n ",
				})

				return err
			},
		},
		{
			name: "query run target_id",
			call: func() error {
				_, err := svc.lrpcQueryRun(ctx, lrpcTestCallID, lrpcQueryRunReq{
					TargetID: " \t\n ",
					Query:    "select 1",
				})

				return err
			},
		},
		{
			name: "query run query",
			call: func() error {
				_, err := svc.lrpcQueryRun(ctx, lrpcTestCallID, lrpcQueryRunReq{
					TargetID: "pg-1",
					Query:    " \t\n ",
				})

				return err
			},
		},
		{
			name: "query results get id",
			call: func() error {
				_, err := svc.lrpcQueryResultsGet(ctx, lrpcTestCallID, lrpcQueryResultsGetReq{ID: " \t\n "})

				return err
			},
		},
		{
			name: "export link query_result_id",
			call: func() error {
				_, err := svc.lrpcQueryResultsExportLink(ctx, lrpcTestCallID, lrpcQueryResultsExportLinkReq{
					QueryResultID: " \t\n ",
					Format:        "json",
				})

				return err
			},
		},
		{
			name: "export link format",
			call: func() error {
				_, err := svc.lrpcQueryResultsExportLink(ctx, lrpcTestCallID, lrpcQueryResultsExportLinkReq{
					QueryResultID: lrpcTestUUID,
					Format:        " \t\n ",
				})

				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.call()
			require.ErrorIs(t, err, errBadInput)
		})
	}
}

func lrpcTestContext() context.Context {
	return context.WithValue(context.Background(), ctxAPITokenUser, structs.User{
		ID:       config.UserID("alice@example.com"),
		Username: "alice",
		Role:     config.RoleUser,
	})
}
