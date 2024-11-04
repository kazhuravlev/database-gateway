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
	ID         string        `json:"id"`
	Type       string        `json:"type"`
	Connection Connection    `json:"connection"`
	Tables     []TargetTable `json:"tables"`
}

type Op string

const (
	OpSelect Op = "select"
	OpInsert Op = "insert"
	OpUpdate Op = "update"
	OpDelete Op = "delete"
)

type ACL struct {
	Op     Op     `json:"op"`
	Target string `json:"target"`
	Tbl    string `json:"tbl"`
	Allow  bool   `json:"allow"`
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Acls     []ACL  `json:"acls"`
}

type Config struct {
	Targets []Target `json:"targets"`
	Users   []User   `json:"users"`
}

type hTable struct {
	target string
	table  string
}

func (c Config) Validate() error {
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

	for _, u := range c.Users {
		for _, acl := range u.Acls {
			key := hTable{
				target: acl.Target,
				table:  acl.Tbl,
			}
			if _, ok := idx[key]; !ok {
				return fmt.Errorf("ACL (%#v) references for not existent table", acl) //nolint:err113
			}
		}
	}

	return nil
}
