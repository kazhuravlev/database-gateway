package config

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

type ACL struct {
	Id     string `json:"id"`
	Target string `json:"target"`
	Select struct {
		Allow  bool     `json:"allow"`
		Tables []string `json:"tables"`
	} `json:"select"`
	Update struct {
		Allow  bool     `json:"allow"`
		Tables []string `json:"tables"`
	} `json:"update"`
	Delete struct {
		Allow  bool     `json:"allow"`
		Tables []string `json:"tables"`
	} `json:"delete"`
}

type User struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Acls     []string `json:"acls"`
}

type Config struct {
	Targets []Target `json:"targets"`
	Acls    []ACL    `json:"acls"`
	Users   []User   `json:"users"`
}
