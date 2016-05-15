package config

import "github.com/getblank/blank-sr/bdb"

type User interface {
	GetId() string
	GetRoles() []string
	GetWorkspace() string
	GetLanguage() string
	Flatten(full bool) bdb.M
}
