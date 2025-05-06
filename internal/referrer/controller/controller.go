package controller

import (
	"context"
	"time"

	"github.com/SakuraBurst/denet/internal/referrer/types"
	"github.com/go-faster/errors"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const defaultRefererReward = 100

type userDatabase interface {
	CreateNewUser(ctx context.Context, user *types.User) error
	GetFullUserInfo(ctx context.Context, userID int) (*types.User, error)
	GetUserByUserName(ctx context.Context, userName string) (*types.User, error)
	RewardUser(ctx context.Context, userID, rewardValue int) error
}

type taskDataBase interface {
	CompleteTask(ctx context.Context, taskID, userID int) (int, error)
	CreateNewTask(ctx context.Context, task *types.Task) (int, error)
	GetTaskById(ctx context.Context, taskID int) (*types.Task, error)
	UpdateTaskReward(ctx context.Context, id, newReward int) error
}

type Controller struct {
	userDatabase userDatabase
	taskDataBase taskDataBase
	jwtSecret    []byte
}

func (c *Controller) CreateNewUser(ctx context.Context, user *types.User) error {
	hashedPass, err := cryptPassword(user.Password)
	if err != nil {
		return errors.Wrap(err, "cryptPassword failed: ")
	}
	user.Password = hashedPass
	err = c.userDatabase.CreateNewUser(ctx, user)
	if err != nil {
		return errors.Wrap(err, "userDatabase.CreateNewUser failed: ")
	}
	return nil
}

func (c *Controller) AuthorizeUser(ctx context.Context, user *types.User) (string, error) {
	user, err := c.userDatabase.GetUserByUserName(ctx, user.UserName)
	if err != nil {
		return "", errors.Wrap(err, "userDatabase.GetUserByUserName failed: ")

	}
	err = bcrypt.CompareHashAndPassword(user.Password, user.Password)
	if err != nil {
		return "", errors.Wrap(err, "bcrypt.CompareHashAndPassword failed: ")
	}

	return createJWT(user.ID)
}

func (c *Controller) GetUserStatus(ctx context.Context, id int) (*types.User, error) {
	user, err := c.userDatabase.GetFullUserInfo(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "serDatabase.GetUserById failed: ")
	}
	return user, nil
}

func (c *Controller) CompleteTask(ctx context.Context, userID int, taskID int) (int, error) {
	return c.taskDataBase.CompleteTask(ctx, taskID, userID)
}

func (c *Controller) Referrer(ctx context.Context, id int) error {
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

// ErrMismatchedHashAndPassword
