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
	"fmt"
	"strings"
)

type UserID string

func (u UserID) S() string {
	return string(u)
}

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func (r Role) S() string {
	return string(r)
}

func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleUser
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

type UsersProviderOIDC struct {
	ClientID            string          `json:"client_id"`
	ClientSecret        string          `json:"client_secret"`
	IssuerURL           string          `json:"issuer_url"`
	RedirectURL         string          `json:"redirect_url"`
	Scopes              []string        `json:"scopes"`
	AccessTokenAudience string          `json:"access_token_audience"`
	RoleClaim           string          `json:"role_claim"            validate:"required"`
	RoleMapping         map[string]Role `json:"role_mapping"          validate:"required"`
}

type FacadeConfig struct {
	Port               int    `json:"port"`
	CookieSecret       string `json:"cookie_secret"`
	UnsafeCORSAllowAll bool   `json:"unsafe_cors_allow_all"`
}

type PolicyConfig struct {
	Path string `json:"path"` // directory with .rego modules; relative paths are resolved from the config file location
}

type Config struct {
	Targets []Target          `json:"targets"`
	Users   UsersProviderOIDC `json:"users"`
	Policy  PolicyConfig      `json:"policy"`
	Facade  FacadeConfig      `json:"facade"`
	Storage PostgresConfig    `json:"storage"`
}

func (c *Config) Validate() error {
	// Check tht each target have table names with schema prefix
	for i := range c.Targets {
		target := c.Targets[i]
		for _, table := range target.Tables {
			if !strings.Contains(table.Table, ".") {
				return fmt.Errorf("use table notation with leading schema. Like 'public.%s'", table.Table) //nolint:err113
			}
		}
	}

	for attrValue, role := range c.Users.RoleMapping {
		if !role.IsValid() {
			return fmt.Errorf("unsupported role %q for users.role_mapping[%q]", role, attrValue) //nolint:err113
		}
	}

	if strings.TrimSpace(c.Policy.Path) == "" {
		return fmt.Errorf("policy.path is required") //nolint:err113
	}

	return nil
}
