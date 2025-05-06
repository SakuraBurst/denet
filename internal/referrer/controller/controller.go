package controller

import (
	"time"

	"github.com/SakuraBurst/denet/internal/referrer/types"
	"github.com/go-faster/errors"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const defaultRefererReward = 100

type userDatabase interface {
	CreateNewUser(user *types.User) error
	GetUserById(userID int) (*types.User, error)
	GetUserByUserName(userName string) (*types.User, error)
	RewardUser(userID, rewardValue int) error
}

type taskDataBase interface {
	CompleteTask(taskID, userID int) (int, error)
	CreateNewTask(task *types.Task) (int, error)
	GetTaskById(taskID int) (*types.Task, error)
	UpdateTaskReward(id, newReward int) error
}

type Controller struct {
	userDatabase userDatabase
	taskDataBase taskDataBase
	jwtSecret    []byte
}

func (c *Controller) CreateNewUser(user *types.User) error {
	hashedPass, err := cryptPassword(user.Password)
	if err != nil {
		return errors.Wrap(err, "cryptPassword failed: ")
	}
	user.Password = hashedPass
	err = c.userDatabase.CreateNewUser(user)
	if err != nil {
		return errors.Wrap(err, "userDatabase.CreateNewUser failed: ")
	}
	return nil
}

func (c *Controller) AuthorizeUser(user *types.User) (string, error) {
	user, err := c.userDatabase.GetUserByUserName(user.UserName)
	if err != nil {
		return "", errors.Wrap(err, "userDatabase.GetUserByUserName failed: ")

	}
	err = bcrypt.CompareHashAndPassword(user.Password, user.Password)
	if err != nil {
		return "", errors.Wrap(err, "bcrypt.CompareHashAndPassword failed: ")
	}

	return createJWT(user.ID)
}

func (c *Controller) GetUserStatus(id int) (*types.User, error) {
	user, err := c.userDatabase.GetUserById(id)
	if err != nil {
		return nil, errors.Wrap(err, "serDatabase.GetUserById failed: ")
	}
	return user, nil
}

func (c *Controller) CompleteTask(userID int, taskID int) (int, error) {
	return c.taskDataBase.CompleteTask(taskID, userID)
}

func (c *Controller) Referrer(id int) error {
	return c.userDatabase.RewardUser(id, defaultRefererReward)
}

func (c *Controller) CreateNewTask(task *types.Task) (int, error) {
	return c.taskDataBase.CreateNewTask(task)
}

func (c *Controller) GetTask(id int) (*types.Task, error) {
	return c.taskDataBase.GetTaskById(id)
}

func (c *Controller) UpdateTaskReward(id, newReward int) error {
	return c.taskDataBase.UpdateTaskReward(id, newReward)
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
