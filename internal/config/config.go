package config

import (
	"fmt"
	"strings"

	"github.com/kazhuravlev/just"
)

type TargetTable struct {
	Table     string   `json:"table"`
	Fields    []string `json:"fields"`
	Sensitive []string `json:"sensitive"`
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

func (t *Target) HasField(tbl, field string) bool {
	for _, f := range t.Tables {
		if f.Table == tbl {
			return just.SliceContainsElem(f.Fields, field)
		}
	}

	return false
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
	for _, t := range c.Targets {
		for _, table := range t.Tables {
			if !strings.Contains(table.Table, ".") {
				return fmt.Errorf("use table notation with leading schema. Like 'public.%s'", table.Table) //nolint:eer113
			}

			key := hTable{
				target: t.ID,
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
				return fmt.Errorf("ACL (%#v) references for not existent table", acl) //nolint:eer113
			}
		}
	}

	return nil
}
