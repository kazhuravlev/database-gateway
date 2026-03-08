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

package facade

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kazhuravlev/database-gateway/internal/app"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
	"github.com/kazhuravlev/just"
	"github.com/kazhuravlev/lrpc/ctypes"
)

var errBadInput = errors.New("bad input")

type lrpcTargetsListResp struct {
	Targets []structs.Server `json:"targets"`
}

type lrpcTargetGetReq struct {
	TargetID string `json:"target_id"`
}

type lrpcTargetGetResp struct {
	Target structs.Server `json:"target"`
}

func (s *Service) lrpcTargetList(ctx context.Context, _ ctypes.ID, _ any) (*lrpcTargetsListResp, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	targets, err := s.opts.app.GetTargets(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
	}

	return &lrpcTargetsListResp{Targets: targets}, nil
}

func (s *Service) lrpcTargetGet(ctx context.Context, _ ctypes.ID, req lrpcTargetGetReq) (*lrpcTargetGetResp, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	targetID := strings.TrimSpace(req.TargetID)
	if targetID == "" {
		return nil, fmt.Errorf("target_id is required: %w", errBadInput)
	}

	target, err := s.opts.app.GetTargetByID(ctx, user, config.TargetID(targetID))
	if err != nil {
		return nil, fmt.Errorf("get target: %w", err)
	}

	return &lrpcTargetGetResp{Target: *target}, nil
}

type Bookmark struct {
	ID       string          `json:"id"`
	TargetID config.TargetID `json:"target_id"`
	Title    string          `json:"title"`
	Query    string          `json:"query"`
}

type lrpcBookmarksListResp struct {
	Bookmarks []Bookmark `json:"bookmarks"`
}

type lrpcBookmarksListReq struct {
	TargetID string `json:"target_id,omitempty"`
}

type lrpcBookmarksAddReq struct {
	TargetID string `json:"target_id"`
	Title    string `json:"title"`
	Query    string `json:"query"`
}

type lrpcBookmarksDeleteReq struct {
	ID string `json:"id"`
}

func (s *Service) lrpcBookmarksList(ctx context.Context, _ ctypes.ID, req lrpcBookmarksListReq) (*lrpcBookmarksListResp, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	targetID := strings.TrimSpace(req.TargetID)

	var bookmarks []structs.Bookmark
	if targetID == "" {
		bookmarks, err = s.opts.app.ListAllBookmarks(ctx, user.ID)
	} else {
		bookmarks, err = s.opts.app.ListBookmarks(ctx, user, config.TargetID(targetID))
	}
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}

	return &lrpcBookmarksListResp{
		Bookmarks: just.SliceMap(bookmarks, func(bookmark structs.Bookmark) Bookmark {
			return Bookmark{
				ID:       bookmark.ID,
				TargetID: bookmark.TargetID,
				Title:    bookmark.Title,
				Query:    bookmark.Query,
			}
		}),
	}, nil
}

func (s *Service) lrpcBookmarksAdd(ctx context.Context, _ ctypes.ID, req lrpcBookmarksAddReq) (*struct{}, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	targetID := strings.TrimSpace(req.TargetID)
	title := strings.TrimSpace(req.Title)
	query := strings.TrimSpace(req.Query)
	if targetID == "" || title == "" || query == "" {
		return nil, fmt.Errorf("target_id, title and query are required: %w", errBadInput)
	}

	if err := s.opts.app.AddBookmark(ctx, user, config.TargetID(targetID), title, query); err != nil {
		return nil, fmt.Errorf("add bookmark: %w", err)
	}

	return &struct{}{}, nil
}

func (s *Service) lrpcBookmarksDelete(
	ctx context.Context,
	_ ctypes.ID,
	req lrpcBookmarksDeleteReq,
) (*struct{}, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	bookmarkID, err := uuid6.ParseStr(strings.TrimSpace(req.ID))
	if err != nil {
		return nil, fmt.Errorf("bad bookmark id: %w", errBadInput)
	}

	if err := s.opts.app.DeleteBookmark(ctx, user.ID, bookmarkID); err != nil {
		return nil, fmt.Errorf("delete bookmark: %w", err)
	}

	return &struct{}{}, nil
}

type Query struct {
	ID        string          `json:"id"`
	TargetID  config.TargetID `json:"target_id"`
	Query     string          `json:"query"`
	CreatedAt string          `json:"created_at"`
}

type lrpcQueriesListResp struct {
	Queries []Query `json:"queries"`
}

type lrpcQueriesListReq struct {
	Limit int64 `json:"limit"`
}

func (s *Service) lrpcQueriesList(ctx context.Context, _ ctypes.ID, req lrpcQueriesListReq) (*lrpcQueriesListResp, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	queries, err := s.opts.app.ListRecentQueries(ctx, user.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("list queries: %w", err)
	}

	return &lrpcQueriesListResp{
		Queries: just.SliceMap(queries, func(query structs.Query) Query {
			return Query{
				ID:        query.ID,
				TargetID:  query.TargetID,
				Query:     query.Query,
				CreatedAt: query.CreatedAt,
			}
		}),
	}, nil
}

type lrpcAdminRequestsListReq struct {
	Page int64 `json:"page"`
}

type AdminRequest struct {
	ID        string          `json:"id"`
	UserID    config.UserID   `json:"user_id"`
	TargetID  config.TargetID `json:"target_id"`
	Query     string          `json:"query"`
	CreatedAt string          `json:"created_at"`
}

type lrpcAdminRequestsListResp struct {
	Requests []AdminRequest `json:"requests"`
	Page     int64          `json:"page"`
	HasPrev  bool           `json:"has_prev"`
	HasNext  bool           `json:"has_next"`
}

func (s *Service) lrpcAdminRequestsList(
	ctx context.Context,
	_ ctypes.ID,
	req lrpcAdminRequestsListReq,
) (*lrpcAdminRequestsListResp, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	const pageSize = 50
	items, hasNext, err := s.opts.app.ListAdminRequests(ctx, user, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list admin requests: %w", err)
	}

	return &lrpcAdminRequestsListResp{
		Requests: just.SliceMap(items, func(item structs.AdminRequest) AdminRequest {
			return AdminRequest{
				ID:        item.ID,
				UserID:    item.UserID,
				TargetID:  item.TargetID,
				Query:     item.Query,
				CreatedAt: item.CreatedAt,
			}
		}),
		Page:    page,
		HasPrev: page > 1,
		HasNext: hasNext,
	}, nil
}

type lrpcProfileGetResp struct {
	ID       config.UserID `json:"id"`
	Username string        `json:"username"`
	Role     config.Role   `json:"role"`
}

func (*Service) lrpcProfileGet(ctx context.Context, _ ctypes.ID, _ any) (*lrpcProfileGetResp, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	return &lrpcProfileGetResp{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

type lrpcQueryRunReq struct {
	TargetID string `json:"target_id"`
	Query    string `json:"query"`
}

type lrpcQueryRunResp struct {
	QueryID string         `json:"query_id"`
	Table   structs.QTable `json:"table"`
}

func (s *Service) lrpcQueryRun(ctx context.Context, _ ctypes.ID, req lrpcQueryRunReq) (*lrpcQueryRunResp, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	targetID := strings.TrimSpace(req.TargetID)
	query := strings.TrimSpace(req.Query)
	if targetID == "" || query == "" {
		return nil, fmt.Errorf("target_id and query are required: %w", errBadInput)
	}

	queryID, table, err := s.opts.app.RunQuery(ctx, user, config.TargetID(targetID), query)
	if err != nil {
		return nil, fmt.Errorf("run query: %w", err)
	}

	return &lrpcQueryRunResp{
		QueryID: queryID.S(),
		Table:   *table,
	}, nil
}

type lrpcQueryResultsGetReq struct {
	ID            string `json:"id,omitempty"`
	QueryResultID string `json:"query_result_id,omitempty"`
}

type lrpcQueryResultsGetResp struct {
	ID        string          `json:"id"`
	UserID    config.UserID   `json:"user_id"`
	TargetID  config.TargetID `json:"target_id"`
	Query     string          `json:"query"`
	CreatedAt string          `json:"created_at"`
	Table     structs.QTable  `json:"table"`
	Meta      structs.QMeta   `json:"meta"`
}

type lrpcQueryResultsExportLinkReq struct {
	QueryResultID string `json:"query_result_id"`
	Format        string `json:"format"`
}

type lrpcQueryResultsExportLinkResp struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

func (s *Service) lrpcQueryResultsGet(
	ctx context.Context,
	_ ctypes.ID,
	req lrpcQueryResultsGetReq,
) (*lrpcQueryResultsGetResp, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	queryResultRaw := strings.TrimSpace(req.QueryResultID)
	if queryResultRaw == "" {
		queryResultRaw = strings.TrimSpace(req.ID)
	}
	queryResultID, err := uuid6.ParseStr(queryResultRaw)
	if err != nil {
		return nil, fmt.Errorf("bad query result id: %w", errBadInput)
	}

	item, err := s.opts.app.GetQueryResults(ctx, user, queryResultID)
	if err != nil {
		return nil, fmt.Errorf("get query result: %w", err)
	}

	return &lrpcQueryResultsGetResp{
		ID:        item.ID,
		UserID:    config.UserID(item.UserID),
		TargetID:  config.TargetID(item.TargetID),
		Query:     item.Query,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
		Table:     item.QTable,
		Meta:      item.Meta,
	}, nil
}

func (s *Service) lrpcQueryResultsExportLink(
	ctx context.Context,
	_ ctypes.ID,
	req lrpcQueryResultsExportLinkReq,
) (*lrpcQueryResultsExportLinkResp, error) {
	user, err := userFromAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	queryResultID, err := uuid6.ParseStr(strings.TrimSpace(req.QueryResultID))
	if err != nil {
		return nil, fmt.Errorf("bad query result id: %w", errBadInput)
	}

	format := strings.TrimSpace(req.Format)
	if !isSupportedExportFormat(format) {
		return nil, fmt.Errorf("bad export format: %w", errBadInput)
	}

	if _, err := s.opts.app.GetQueryResults(ctx, user, queryResultID); err != nil {
		return nil, fmt.Errorf("get query result: %w", err)
	}

	expiresAt := time.Now().Add(3 * time.Second)
	token, err := s.buildQueryResultsExportToken(user.ID, queryResultID, format, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("build export token: %w", err)
	}

	return &lrpcQueryResultsExportLinkResp{
		URL:       "/api/v1/query-results/export/" + token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func userFromAPIToken(ctx context.Context) (structs.User, error) {
	user, ok := ctx.Value(ctxAPITokenUser).(structs.User)
	if !ok {
		return structs.User{}, fmt.Errorf("authenticate api token: %w", app.ErrForbidden)
	}

	return user, nil
}
