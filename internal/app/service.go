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

type Service struct {
	opts Options

	connsMu *sync.RWMutex
	conns   map[config.TargetID]*pgxpool.Pool
	// NOTE: can be nil (depends on [Options.cfg.Users.Provider])
	oauthCfg     *oauth2.Config
	oidcProvider *oidc.Provider
}

func New(opts Options) (*Service, error) { //nolint:gocritic
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("bad configuration: %w", err)
	}

	var oidcProvider *oidc.Provider
	var oauthCfg *oauth2.Config
	if oidcCfg, ok := opts.users.Provider.(config.UsersProviderOIDC); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //nolint:mnd
		defer cancel()

		provider, err := oidc.NewProvider(ctx, oidcCfg.IssuerURL)
		if err != nil {
			return nil, fmt.Errorf("init provider: %w", err)
		}

		oauthCfg = &oauth2.Config{
			ClientID:     oidcCfg.ClientID,
			ClientSecret: oidcCfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  oidcCfg.RedirectURL,
			Scopes:       append([]string{oidc.ScopeOpenID}, oidcCfg.Scopes...),
		}
		oidcProvider = provider
	}

	return &Service{
		opts:         opts,
		connsMu:      new(sync.RWMutex),
		conns:        make(map[config.TargetID]*pgxpool.Pool),
		oidcProvider: oidcProvider,
		oauthCfg:     oauthCfg,
	}, nil
}

func (s *Service) AuthUser(_ context.Context, username, password string) (*structs.User, error) {
	user, err := s.findUser(func(user config.User) bool {
		return user.Username == username && user.Password == password
	})
	if err != nil {
		return nil, err
	}

	return &structs.User{
		ID:       user.ID,
		Username: user.Username,
	}, nil
}

// GetTargets return targets that available for this user id.
func (s *Service) GetTargets(_ context.Context, uID config.UserID) ([]structs.Server, error) {
	availableTargets := just.SliceFilter(s.opts.targets, func(target config.Target) bool {
		return s.opts.acls.Allow(rules.ByUserID(uID.S()), rules.ByTargetID(target.ID.S()))
	})

	servers := just.SliceMap(availableTargets, adaptTarget)

	return servers, nil
}

func (s *Service) GetTargetByID(ctx context.Context, uID config.UserID, tID config.TargetID) (*structs.Server, error) {
	res, err := s.getTargetByID(ctx, uID, tID)
	if err != nil {
		return nil, fmt.Errorf("get target: %w", err)
	}

	return just.Pointer(adaptTarget(*res)), nil
}

func (s *Service) RunQuery(ctx context.Context, userID config.UserID, srvID config.TargetID, query string) (uuid6.UUID, *structs.QTable, error) {
	srv, err := s.getTargetByID(ctx, userID, srvID)
	if err != nil {
		return uuid6.Nil(), nil, fmt.Errorf("get target by id: %w", err)
	}

	haveAccess := func(vec validator.Vec) bool {
		return s.opts.acls.Allow(
			rules.ByUserID(userID.S()),
			rules.ByTargetID(srvID.S()),
			rules.ByOp(vec.Op.S()),
			rules.ByTable(vec.Tbl),
		)
	}

	if err := validator.IsAllowed(srv.Tables, haveAccess, query); err != nil {
		log.Error("err", err.Error())

		return uuid6.Nil(), nil, fmt.Errorf("preflight check: %w", err)
	}

	conn, err := s.getConnection(ctx, *srv)
	if err != nil {
		return uuid6.Nil(), nil, fmt.Errorf("get connection by id: %w", err)
	}

	queryStart := time.Now()
	res, err := conn.Query(ctx, query)
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

	bb, err := json.Marshal(qTable)
	if err != nil {
		return uuid6.Nil(), nil, fmt.Errorf("marshal qtable: %w", err)
	}

	req := storage.InsertQueryResultsReq{
		ID:        uuid6.New(),
		UserID:    userID,
		CreatedAt: queryStart,
		Query:     query,
		Response:  bb,
	}
	if err := s.opts.storage.InsertQueryResults(s.opts.storage.Conn(ctx), req); err != nil {
		return uuid6.Nil(), nil, fmt.Errorf("insert query results: %w", err)
	}

	return req.ID, &qTable, nil
}

func (s *Service) AuthType() config.AuthType {
	return s.opts.users.Provider.Type()
}

func (s *Service) InitOIDC(_ context.Context) (string, error) {
	if s.oauthCfg == nil {
		return "", errors.New("not available for this provider") //nolint:err113
	}

	state := just.Must(uuid.NewUUID()).String()

	return s.oauthCfg.AuthCodeURL(state), nil
}

func (s *Service) CompleteOIDC(ctx context.Context, code string) (*structs.User, time.Time, error) {
	if s.oauthCfg == nil {
		return nil, time.Time{}, errors.New("not available for this provider") //nolint:err113
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

	expiry := token.Expiry
	if expiry.IsZero() {
		expiry = time.Now().Add(15 * time.Minute) //nolint:mnd
	}

	return &structs.User{
		ID:       config.UserID(claims.Email),
		Username: just.If(claims.PreferredUsername != "", claims.PreferredUsername, claims.Email),
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

	var qTable structs.QTable
	if err := json.Unmarshal(res.Response, &qTable); err != nil {
		return nil, fmt.Errorf("unmarshal query results: %w", err)
	}

	return &QueryResults{
		CreatedAt: res.CreatedAt,
		Query:     res.Query,
		QTable:    qTable,
	}, nil
}
