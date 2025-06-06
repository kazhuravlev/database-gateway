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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kazhuravlev/database-gateway/internal/app/rules"
	"github.com/kazhuravlev/just"
)

type AuthType string

const (
	AuthTypeConfig AuthType = "config"
	AuthTypeOIDC   AuthType = "oidc"
)

type UserID string

func (u UserID) S() string {
	return string(u)
}

type TargetID string

func (t TargetID) S() string {
	return string(t)
}

type Op string

const (
	OpSelect Op = "select"
	OpInsert Op = "insert"
	OpUpdate Op = "update"
	OpDelete Op = "delete"
)

func (op Op) S() string {
	return string(op)
}

type PostgresConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Database    string `json:"database"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	UseSSL      bool   `json:"use_ssl"`
	MaxPoolSize int    `json:"max_pool_size"`
}

type TargetTable struct {
	Table  string   `json:"table"`
	Fields []string `json:"fields"`
}

type Connection struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	Password    string `json:"password"`
	DB          string `json:"db"`
	UseSSL      bool   `json:"use_ssl"`
	MaxPoolSize int    `json:"max_pool_size"`
}

type Target struct {
	ID            TargetID      `json:"id"`
	Description   string        `json:"description"`
	Tags          []string      `json:"tags"`
	Type          string        `json:"type"`
	Connection    Connection    `json:"connection"`
	DefaultSchema string        `json:"default_schema"`
	Tables        []TargetTable `json:"tables"`
}

type User struct {
	ID       UserID `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type IProvider interface {
	isProviderConfiguration()
	Type() AuthType
}

type UsersConfig struct {
	Provider IProvider
}

func (u *UsersConfig) UnmarshalJSON(data []byte) error {
	var cfg struct {
		Provider string `json:"provider"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("unmarshal users config: %w", err)
	}

	switch AuthType(cfg.Provider) {
	default:
		return errors.New("unknown users provider") //nolint:err113
	case AuthTypeConfig:
		var res struct {
			Configuration UsersProviderConfig `json:"configuration"`
		}
		if err := json.Unmarshal(data, &res); err != nil {
			return fmt.Errorf("unmarshal users config: %w", err)
		}
		*u = UsersConfig{
			Provider: res.Configuration,
		}
	case AuthTypeOIDC:
		var res struct {
			Configuration UsersProviderOIDC `json:"configuration"`
		}
		if err := json.Unmarshal(data, &res); err != nil {
			return fmt.Errorf("unmarshal users oidc: %w", err)
		}
		*u = UsersConfig{
			Provider: res.Configuration,
		}
	}

	return nil
}

type UsersProviderConfig []User

func (UsersProviderConfig) Type() AuthType {
	return AuthTypeConfig
}

func (UsersProviderConfig) isProviderConfiguration() {}

type UsersProviderOIDC struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	IssuerURL    string   `json:"issuer_url"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
}

func (UsersProviderOIDC) Type() AuthType {
	return AuthTypeOIDC
}

func (UsersProviderOIDC) isProviderConfiguration() {}

type FacadeConfig struct {
	Port         int    `json:"port"`
	CookieSecret string `json:"cookie_secret"`
}

type Config struct {
	Targets []Target       `json:"targets"`
	Users   UsersConfig    `json:"users"`
	ACLs    []rules.ACL    `json:"acls"`
	Facade  FacadeConfig   `json:"facade"`
	Storage PostgresConfig `json:"storage"`
}

func (c *Config) Validate() error { //nolint:cyclop
	type hTable struct {
		target TargetID
		table  string
	}

	// Check tht each target have table names with schema prefix
	idx := make(map[hTable]struct{}, len(c.Targets)*2)
	for i := range c.Targets {
		target := c.Targets[i]
		for _, table := range target.Tables {
			if !strings.Contains(table.Table, ".") {
				return fmt.Errorf("use table notation with leading schema. Like 'public.%s'", table.Table) //nolint:err113
			}

			key := hTable{
				target: target.ID,
				table:  table.Table,
			}
			idx[key] = struct{}{}
		}
	}

	// Check that all acls linked with exists targets
	for _, acl := range c.ACLs {
		if acl.Target == rules.Star || acl.Tbl == rules.Star {
			continue
		}

		key := hTable{
			target: TargetID(acl.Target),
			table:  acl.Tbl,
		}
		if _, ok := idx[key]; !ok {
			return fmt.Errorf("ACL (%#v) references for not existent table", acl) //nolint:err113
		}
	}

	// Check that acl relates to exists user (for config-based provider)
	if users, ok := c.Users.Provider.(UsersProviderConfig); ok {
		userMap := just.Slice2MapFn(users, func(_ int, user User) (UserID, struct{}) {
			return user.ID, struct{}{}
		})

		for _, acl := range c.ACLs {
			if !just.MapContainsKey(userMap, UserID(acl.User)) {
				return fmt.Errorf("ACL (%#v) targets to unknown user", acl) //nolint:err113
			}
		}
	}

	return nil
}
