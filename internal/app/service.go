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

package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kazhuravlev/database-gateway/internal/app/rules"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/storage"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/kazhuravlev/just"
	"github.com/labstack/gommon/log"
	"golang.org/x/oauth2"
)

var ErrNotFound = errors.New("not found")

type storedQueryResultPayload struct {
	Table structs.QTable `json:"table"`
	Meta  structs.QMeta  `json:"meta"`
}

type Service struct {
	opts Options

	connsMu      *sync.RWMutex
	conns        map[config.TargetID]*pgxpool.Pool
	oauthCfg     *oauth2.Config
	oidcProvider *oidc.Provider
}

func New(opts Options) (*Service, error) { //nolint:gocritic
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("bad configuration: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //nolint:mnd
	defer cancel()

	oidcCfg := opts.users
	if len(oidcCfg.RoleMapping) == 0 {
		return nil, errors.New("no role mappings defined") //nolint:err113
	}

	oidcProvider, err := oidc.NewProvider(ctx, oidcCfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("init provider: %w", err)
	}

	oauthCfg := &oauth2.Config{
		ClientID:     oidcCfg.ClientID,
		ClientSecret: oidcCfg.ClientSecret,
		Endpoint:     oidcProvider.Endpoint(),
		RedirectURL:  oidcCfg.RedirectURL,
		Scopes:       append([]string{oidc.ScopeOpenID}, oidcCfg.Scopes...),
	}

	return &Service{
		opts:         opts,
		connsMu:      new(sync.RWMutex),
		conns:        make(map[config.TargetID]*pgxpool.Pool),
		oidcProvider: oidcProvider,
		oauthCfg:     oauthCfg,
	}, nil
}

// GetTargets return targets that available for this user.
func (s *Service) GetTargets(_ context.Context, user structs.User) ([]structs.Server, error) {
	subjects := userSubjects(user)
	availableTargets := just.SliceFilter(s.opts.targets, func(target config.Target) bool {
		return s.opts.acls.Allow(rules.BySubjects(subjects...), rules.ByTargetID(target.ID.S()))
	})

	servers := just.SliceMap(availableTargets, adaptTarget)

	return servers, nil
}

func (s *Service) GetTargetByID(ctx context.Context, user structs.User, tID config.TargetID) (*structs.Server, error) {
	res, _, err := s.getTargetByID(ctx, user, tID)
	if err != nil {
		return nil, fmt.Errorf("get target: %w", err)
	}

	return just.Pointer(adaptTarget(*res)), nil //nolint:modernize // false positive
}

func (s *Service) RunQuery(
	ctx context.Context,
	user structs.User,
	srvID config.TargetID,
	query string,
) (uuid6.UUID, *structs.QTable, error) {
	fullRoundTripStartedAt := time.Now()

	srv, schema, err := s.getTargetByID(ctx, user, srvID)
	if err != nil {
		return uuid6.Nil(), nil, fmt.Errorf("get target by id: %w", err)
	}
	subjects := userSubjects(user)

	haveAccess := func(vec validator.Vec) bool {
		return s.opts.acls.Allow(
			rules.BySubjects(subjects...),
			rules.ByTargetID(srvID.S()),
			rules.ByOp(vec.Op.S()),
			rules.ByTable(vec.Tbl),
		)
	}

	parsingStartedAt := time.Now()
	vectors, err := validator.MakeVectors(query)
	parsingDuration := time.Since(parsingStartedAt)
	if err != nil {
		log.Error("err", err.Error())

		return uuid6.Nil(), nil, fmt.Errorf("preflight check: make vectors: %w", err)
	}

	if err := validator.ValidateSchema(vectors, schema); err != nil {
		log.Error("err", err.Error())

		return uuid6.Nil(), nil, fmt.Errorf("preflight check: validate schema: %w", err)
	}

	if err := validator.ValidateAccess(vectors, haveAccess); err != nil {
		log.Error("err", err.Error())

		return uuid6.Nil(), nil, fmt.Errorf("preflight check: validate access: %w", err)
	}

	conn, err := s.getConnection(ctx, *srv)
	if err != nil {
		return uuid6.Nil(), nil, fmt.Errorf("get connection by id: %w", err)
	}

	queryStartedAt := time.Now()
	res, err := conn.Query(ctx, query)
	networkRoundTripDuration := time.Since(queryStartedAt)
	if err != nil {
		return uuid6.Nil(), nil, fmt.Errorf("query: %w", err)
	}

	rows, err := pgx.CollectRows(res, func(row pgx.CollectableRow) ([]any, error) {
		return row.Values()
	})
	if err != nil {
		return uuid6.Nil(), nil, fmt.Errorf("collect rowsL %w", err)
	}

	cols := just.SliceMap(res.FieldDescriptions(), func(fd pgconn.FieldDescription) string {
		return fd.Name
	})

	qTable := structs.QTable{
		Headers: cols,
		Rows: just.SliceMap(rows, func(row []any) []string {
			return just.SliceMap(row, adaptPgType)
		}),
	}

	meta := structs.QMeta{
		ExecutionTimeMS:    time.Since(fullRoundTripStartedAt).Milliseconds(),
		ParsingTimeMS:      parsingDuration.Milliseconds(),
		NetworkRoundTripMS: networkRoundTripDuration.Milliseconds(),
		RowsCount:          len(rows),
		ColumnsCount:       len(cols),
		VectorsCount:       len(vectors),
	}

	buf, err := json.Marshal(storedQueryResultPayload{
		Table: qTable,
		Meta:  meta,
	})
	if err != nil {
		return uuid6.Nil(), nil, fmt.Errorf("marshal qtable: %w", err)
	}

	req := storage.InsertQueryResultsReq{
		ID:        uuid6.New(),
		UserID:    user.ID,
		TargetID:  srvID,
		CreatedAt: queryStartedAt,
		Query:     query,
		Response:  buf,
	}
	if err := s.opts.storage.InsertQueryResults(s.opts.storage.Conn(ctx), req); err != nil {
		return uuid6.Nil(), nil, fmt.Errorf("insert query results: %w", err)
	}

	return req.ID, &qTable, nil
}

func (s *Service) InitOIDC(_ context.Context) (string, string, error) { //nolint:gocritic
	state := just.Must(uuid.NewUUID()).String()

	return s.oauthCfg.AuthCodeURL(state), state, nil
}

func (s *Service) CompleteOIDC( //nolint:cyclop
	ctx context.Context,
	code, expectedState, receivedState string,
) (*structs.User, time.Time, error) {
	// Validate state parameter to prevent CSRF attacks
	if expectedState != receivedState || expectedState == "" {
		return nil, time.Time{}, errors.New("invalid state parameter - possible CSRF attack") //nolint:err113
	}

	token, err := s.oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("exchange token: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, time.Time{}, errors.New("id_token not found in response") //nolint:err113
	}

	idToken, err := s.oidcProvider.Verifier(&oidc.Config{ClientID: s.oauthCfg.ClientID}).Verify(ctx, rawIDToken) //nolint:exhaustruct
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("verify id_token: %w", err)
	}

	var claims struct {
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, time.Time{}, fmt.Errorf("parse id_token claims: %w", err)
	}

	if claims.Email == "" {
		return nil, time.Time{}, errors.New("email claim is required in id_token") //nolint:err113
	}

	var rawClaims map[string]json.RawMessage
	if err := idToken.Claims(&rawClaims); err != nil {
		return nil, time.Time{}, fmt.Errorf("parse raw id_token claims: %w", err)
	}

	role, err := resolveUserRole(rawClaims, s.opts.users.RoleClaim, s.opts.users.RoleMapping)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("resolve role: %w", err)
	}

	expiry := token.Expiry
	if expiry.IsZero() {
		expiry = time.Now().Add(15 * time.Minute) //nolint:mnd
	}

	return &structs.User{
		ID:       config.UserID(claims.Email),
		Username: just.If(claims.PreferredUsername != "", claims.PreferredUsername, claims.Email),
		Role:     role,
	}, expiry, nil
}

func (s *Service) GetQueryResults(ctx context.Context, uid config.UserID, qid uuid6.UUID) (*QueryResults, error) {
	res, err := s.opts.storage.GetQueryResults(s.opts.storage.Conn(ctx), uid, qid)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, fmt.Errorf("unknown result id: %w", ErrNotFound)
		}

		return nil, fmt.Errorf("get query results: %w", err)
	}

	var payload storedQueryResultPayload
	if err := json.Unmarshal(res.Response, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal query results: %w", err)
	}

	return &QueryResults{
		CreatedAt: res.CreatedAt,
		Query:     res.Query,
		QTable:    payload.Table,
		Meta:      payload.Meta,
	}, nil
}

func (s *Service) AddBookmark(
	ctx context.Context,
	user structs.User,
	targetID config.TargetID,
	title string,
	query string,
) error {
	trimmedTitle := strings.TrimSpace(title)
	trimmedQuery := strings.TrimSpace(query)
	if trimmedTitle == "" || trimmedQuery == "" {
		return errors.New("title and query are required") //nolint:err113
	}

	if _, _, err := s.getTargetByID(ctx, user, targetID); err != nil {
		return fmt.Errorf("validate target access: %w", err)
	}

	req := storage.InsertBookmarkReq{
		ID:        uuid6.New(),
		UserID:    user.ID,
		TargetID:  targetID,
		Title:     trimmedTitle,
		Query:     trimmedQuery,
		CreatedAt: time.Now(),
	}
	if err := s.opts.storage.InsertBookmark(s.opts.storage.Conn(ctx), req); err != nil {
		return fmt.Errorf("insert bookmark: %w", err)
	}

	return nil
}

func (s *Service) DeleteBookmark(ctx context.Context, uid config.UserID, bookmarkID uuid6.UUID) error {
	if err := s.opts.storage.DeleteBookmark(s.opts.storage.Conn(ctx), uid, bookmarkID); err != nil {
		return fmt.Errorf("delete bookmark: %w", err)
	}

	return nil
}

func (s *Service) ListBookmarks(ctx context.Context, user structs.User, targetID config.TargetID) ([]structs.Bookmark, error) {
	if _, _, err := s.getTargetByID(ctx, user, targetID); err != nil {
		return nil, fmt.Errorf("validate target access: %w", err)
	}

	items, err := s.opts.storage.ListBookmarks(s.opts.storage.Conn(ctx), user.ID, targetID)
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}

	out := just.SliceMap(items, func(item storage.Bookmark) structs.Bookmark {
		return structs.Bookmark{
			ID:       item.ID.S(),
			TargetID: item.TargetID,
			Title:    item.Title,
			Query:    item.Query,
		}
	})

	return out, nil
}

func (s *Service) ListAllBookmarks(ctx context.Context, uid config.UserID) ([]structs.Bookmark, error) {
	items, err := s.opts.storage.ListBookmarksByUser(s.opts.storage.Conn(ctx), uid)
	if err != nil {
		return nil, fmt.Errorf("list bookmarks by user: %w", err)
	}

	out := just.SliceMap(items, func(item storage.Bookmark) structs.Bookmark {
		return structs.Bookmark{
			ID:       item.ID.S(),
			TargetID: item.TargetID,
			Title:    item.Title,
			Query:    item.Query,
		}
	})

	return out, nil
}

func (s *Service) ListRecentQueries(ctx context.Context, uid config.UserID, limit int64) ([]structs.RecentQuery, error) {
	if limit <= 0 {
		limit = 50
	}

	items, err := s.opts.storage.ListQueryResultsByUser(s.opts.storage.Conn(ctx), uid, limit)
	if err != nil {
		return nil, fmt.Errorf("list query results by user: %w", err)
	}

	out := make([]structs.RecentQuery, 0, len(items))
	for _, item := range items {
		var payload storedQueryResultPayload
		if err := json.Unmarshal(item.Response, &payload); err != nil {
			continue
		}

		out = append(out, structs.RecentQuery{
			ID:        item.ID.S(),
			TargetID:  item.TargetID,
			Query:     item.Query,
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return out, nil
}

func resolveUserRole(claims map[string]json.RawMessage, roleClaim string, roleMapping map[string]config.Role) (config.Role, error) {
	claimValues, err := getClaimValues(claims, roleClaim)
	if err != nil {
		return "", err
	}

	for _, claimValue := range claimValues {
		if role, ok := roleMapping[claimValue]; ok {
			return role, nil
		}
	}

	return "", errors.New("no role found") //nolint:err113
}

func getClaimValues(claims map[string]json.RawMessage, claimName string) ([]string, error) {
	rawClaim, ok := claims[claimName]
	if !ok {
		return nil, errors.New("claim not found") //nolint:err113
	}

	var claimValues []string
	if err := json.Unmarshal(rawClaim, &claimValues); err != nil {
		return nil, fmt.Errorf("claim (%q) must be []string type", claimName) //nolint:err113
	}

	return claimValues, nil
}

func userSubjects(user structs.User) []string {
	return []string{
		rules.UserPrincipal(user.ID.S()),
		rules.RolePrincipal(user.Role.S()),
	}
}
