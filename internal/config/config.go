package config

import "github.com/kazhuravlev/just"

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
	Db          string `json:"db"`
	UseSSL      bool   `json:"use_ssl"`
	MaxPoolSize int    `json:"max_pool_size"`
}

type Target struct {
	Id         string        `json:"id"`
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

type ACL struct {
	Op     string `json:"op"`
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
