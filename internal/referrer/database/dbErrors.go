package database

import "github.com/go-faster/errors"

var ErrUserAlreadyExist = errors.New("user already exist")
var ErrUserNotExist = errors.New("user not exist")
