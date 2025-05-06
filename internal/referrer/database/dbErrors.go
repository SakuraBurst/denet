package database

import "github.com/go-faster/errors"

var ErrUserAlreadyExist = errors.New("user already exist")
var ErrUserNotExist = errors.New("user not exist")
var ErrTaskNotExist = errors.New("task not exist")
var ErrTaskAlreadyExist = errors.New("task already exist")
var ErrAlreadyCompletedTask = errors.New("task already completed")
