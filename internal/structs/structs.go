package structs

import "github.com/kazhuravlev/database-gateway/internal/config"

type Server struct {
	ID     string
	Type   string
	Tables []config.TargetTable
}

type User struct {
	Username string
}
