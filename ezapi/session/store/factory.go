package store

import (
	ginSession "github.com/gin-contrib/sessions"
)

func NewStore(store string, maxAge int, keyPairs ...[]byte) Store {
	if store == "mongo" {
		return NewMongoStore(maxAge, keyPairs...)
	}
	return nil
}

type Store interface {
	ginSession.Store
	EncodeToken(name, id string) (string, error)
}
