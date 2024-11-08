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
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
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
	if oidcCfg, ok := opts.cfg.Users.Provider.(config.UsersProviderOIDC); ok {
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

func (s *Service) findUser(fn func(user config.User) bool) (*config.User, error) {
	users, ok := s.opts.cfg.Users.Provider.(config.UsersProviderConfig)
	if !ok {
		return nil, errors.New("not implemented") //nolint:err113
	}

	user := just.SliceFindFirst(users, func(_ int, user config.User) bool {
		return fn(user)
	})

	if user, ok := user.ValueOk(); ok {
		return &user, nil
	}

	return nil, fmt.Errorf("user not exists: %w", ErrNotFound)
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

func (s *Service) GetUserByID(_ context.Context, id config.UserID) (*config.User, error) {
	user, err := s.findUser(func(user config.User) bool {
		return user.ID == id
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetTargets return targets that available for this user id.
func (s *Service) GetTargets(_ context.Context, uID config.UserID) ([]structs.Server, error) {
	userACLs := just.SliceFilter(s.opts.cfg.ACLs, func(acl config.ACL) bool {
		// Filter acls that related to user
		return acl.User == uID && acl.Allow
	})

	targets := just.Slice2MapFn(userACLs, func(_ int, acl config.ACL) (config.TargetID, struct{}) {
		return acl.Target, struct{}{}
	})

	availableTargets := just.SliceFilter(s.opts.cfg.Targets, func(target config.Target) bool {
		return just.MapContainsKey(targets, target.ID)
	})

	servers := just.SliceMap(availableTargets, func(t config.Target) structs.Server {
		return structs.Server{
			ID:          t.ID,
			Description: t.Description,
			Tags:        t.Tags,
			Type:        t.Type,
			Tables:      t.Tables,
		}
	})

	return servers, nil
}

func (s *Service) GetTargetByID(ctx context.Context, uID config.UserID, tID config.TargetID) (*config.Target, error) {
	for i := range s.opts.cfg.Targets {
		target := s.opts.cfg.Targets[i]
		if target.ID == tID {
			acls := just.SliceFilter(s.FilterACLs(ctx, uID, tID), func(acl config.ACL) bool {
				return acl.Allow
			})

			if len(acls) == 0 {
				return nil, fmt.Errorf("target not found: %w", ErrNotFound)
			}

			return &target, nil
		}
	}

	return nil, fmt.Errorf("target not found: %w", ErrNotFound)
}

func (s *Service) FilterACLs(_ context.Context, uID config.UserID, tID config.TargetID) []config.ACL {
	return just.SliceFilter(s.opts.cfg.ACLs, func(acl config.ACL) bool {
		return acl.Target == tID && acl.User == uID
	})
}

func (s *Service) RunQuery(ctx context.Context, userID config.UserID, srvID config.TargetID, query string) (*structs.QTable, error) {
	srv, err := s.GetTargetByID(ctx, userID, srvID)
	if err != nil {
		return nil, fmt.Errorf("get target by id: %w", err)
	}

	acls := s.FilterACLs(ctx, userID, srvID)

	if err := validator.IsAllowed(srv.Tables, acls, query); err != nil {
		log.Error("err", err.Error())

		return nil, fmt.Errorf("preflight check: %w", err)
	}

	conn, err := s.getConnectionByID(ctx, *srv)
	if err != nil {
		return nil, fmt.Errorf("get connection by id: %w", err)
	}

	res, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	rows, err := pgx.CollectRows(res, func(row pgx.CollectableRow) ([]any, error) {
		return row.Values()
	})
	if err != nil {
		return nil, fmt.Errorf("collect rowsL %w", err)
	}

	cols := just.SliceMap(res.FieldDescriptions(), func(fd pgconn.FieldDescription) string {
		return fd.Name
	})

	return &structs.QTable{
		Headers: cols,
		Rows: just.SliceMap(rows, func(row []any) []string {
			return just.SliceMap(row, adaptPgType)
		}),
	}, nil
}

func adaptPgType(val any) string {
	switch val := val.(type) {
	default:
		return fmt.Sprint(val)
	case pgtype.Numeric:
		// TODO: is that really best solution?
		res, err := val.MarshalJSON()
		if err != nil {
			return "--bad payload--"
		}

		return string(bytes.Trim(res, `"`))
	}
}

func (s *Service) getConnectionByID(ctx context.Context, target config.Target) (*pgxpool.Pool, error) { //nolint:gocritic
	{
		s.connsMu.RLock()
		pool, ok := s.conns[target.ID]
		s.connsMu.RUnlock()

		if ok {
			return pool, nil
		}
	}

	s.opts.logger.Info("connect to target", slog.String("target", string(target.ID)))
	pgCfg := target.Connection

	urlExample := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		pgCfg.User,
		pgCfg.Password,
		pgCfg.Host,
		pgCfg.Port,
		pgCfg.DB,
		just.If(pgCfg.UseSSL, "enable", "disable"),
	)
	dbpool, err := pgxpool.New(ctx, urlExample)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}
	// TODO: we cann disconnect on shutdown. But this is not so important.
	// dbpool.Close()

	s.connsMu.Lock()
	s.conns[target.ID] = dbpool
	s.connsMu.Unlock()

	return dbpool, nil
}

func (s *Service) AuthType() config.AuthType {
	return s.opts.cfg.Users.Provider.Type()
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
