package controller

import (
	"context"
	"time"

	"github.com/SakuraBurst/denet/internal/referrer/config"
	"github.com/SakuraBurst/denet/internal/referrer/types"
	"github.com/go-faster/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const defaultRefererReward = 100

type userDatabase interface {
	CreateNewUser(ctx context.Context, user *types.UserRequest, referrerCode string) error
	GetFullUserInfo(ctx context.Context, userID int) (*types.User, error)
	GetUserByUserName(ctx context.Context, userName string) (*types.User, error)
	GetUserByReferrerCode(ctx context.Context, referrerCode string) (*types.User, error)
	RewardUser(ctx context.Context, userID, rewardValue int) error
}

type taskToUserDatabase interface {
	CompleteTask(ctx context.Context, taskID, userID int) (int, error)
}

type taskDataBase interface {
	CreateNewTask(ctx context.Context, task *types.Task) (int, error)
	GetTaskById(ctx context.Context, taskID int) (*types.Task, error)
	UpdateTaskReward(ctx context.Context, id, newReward int) error
}

type Controller struct {
	userDatabase       userDatabase
	taskDataBase       taskDataBase
	taskToUserDatabase taskToUserDatabase
	jwtSecret          []byte
	databaseClose      func() error
}

func NewController(cfg *config.Config, u userDatabase, t taskDataBase, ttu taskToUserDatabase, dbClose func() error) *Controller {
	return &Controller{
		userDatabase:       u,
		taskDataBase:       t,
		taskToUserDatabase: ttu,
		jwtSecret:          []byte(cfg.JWTSecret),
		databaseClose:      dbClose,
	}
}

func (c *Controller) CreateNewUser(ctx context.Context, user *types.UserRequest) error {
	hashedPass, err := cryptPassword([]byte(user.Password))
	if err != nil {
		return errors.Wrap(err, "cryptPassword failed: ")
	}
	user.Password = string(hashedPass)
	err = c.userDatabase.CreateNewUser(ctx, user, uuid.New().String())
	if err != nil {
		return errors.Wrap(err, "userDatabase.CreateNewUser failed: ")
	}
	return nil
}

func (c *Controller) AuthorizeUser(ctx context.Context, user *types.UserRequest) (string, error) {
	foundUser, err := c.userDatabase.GetUserByUserName(ctx, user.UserName)
	if err != nil {
		return "", errors.Wrap(err, "userDatabase.GetUserByUserName failed: ")

	}
	err = bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(user.Password))
	if err != nil {
		return "", errors.Wrap(err, "bcrypt.CompareHashAndPassword failed: ")
	}

	return createJWT(foundUser.ID)
}

func (c *Controller) GetUserStatus(ctx context.Context, id int) (*types.User, error) {
	user, err := c.userDatabase.GetFullUserInfo(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "userDatabase.GetUserById failed: ")
	}
	return user, nil
}

func (c *Controller) CompleteTask(ctx context.Context, userID int, taskID int) (int, error) {
	return c.taskToUserDatabase.CompleteTask(ctx, taskID, userID)
}

func (c *Controller) Referrer(ctx context.Context, id int, referrerCode string) error {
	codeOwner, err := c.userDatabase.GetUserByReferrerCode(ctx, referrerCode)
	if err != nil {
		return errors.Wrap(err, "userDatabase.GetUserByReferrerCode failed: ")
	}
	// поощряем и того кто ввел код, и того чей код был введен
	err = c.userDatabase.RewardUser(ctx, codeOwner.ID, defaultRefererReward)
	if err != nil {
		return errors.Wrap(err, "userDatabase.RewardUser failed: ")
	}
	return c.userDatabase.RewardUser(ctx, id, defaultRefererReward)
}

func (c *Controller) CreateNewTask(ctx context.Context, task *types.Task) (int, error) {
	return c.taskDataBase.CreateNewTask(ctx, task)
}

func (c *Controller) GetTask(ctx context.Context, id int) (*types.Task, error) {
	return c.taskDataBase.GetTaskById(ctx, id)
}

func (c *Controller) UpdateTaskReward(ctx context.Context, id, newReward int) error {
	return c.taskDataBase.UpdateTaskReward(ctx, id, newReward)
}

func (c *Controller) Close() error {
	return c.databaseClose()
}

func createJWT(id int) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = id
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	return token.SignedString([]byte("secret"))

}

func cryptPassword(pass []byte) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword(pass, bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(err, "bcrypt.GenerateFromPassword failed: ")
	}
	return hash, nil
}
