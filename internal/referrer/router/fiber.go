package router

import (
	"context"
	"net/http"
	"strconv"

	"github.com/SakuraBurst/denet/internal/referrer/config"
	"github.com/SakuraBurst/denet/internal/referrer/database"
	"github.com/SakuraBurst/denet/internal/referrer/router/middleware"
	"github.com/SakuraBurst/denet/internal/referrer/types"
	"github.com/go-faster/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type controller interface {
	CreateNewUser(ctx context.Context, user *types.UserRequest) error
	AuthorizeUser(ctx context.Context, user *types.UserRequest) (string, error)
	GetUserStatus(ctx context.Context, id int) (*types.FullUser, error)
	CompleteTask(ctx context.Context, userID int, taskID int) (int, error)
	Referrer(ctx context.Context, id int, referrerCode string) error
	CreateNewTask(ctx context.Context, task *types.Task) (int, error)
	GetTask(ctx context.Context, id int) (*types.Task, error)
	UpdateTaskReward(ctx context.Context, id, newReward int) error
	GetAllTasks(ctx context.Context) ([]*types.Task, error)
	GetTopUsers(ctx context.Context) ([]*types.User, error)
	Close() error
}

type HttpRouter struct {
	controller controller
	*fiber.App
	appLogger *zap.Logger
	httpPort  string
}

const internalServerErrorMessage = "Произошла ошибка на сервере"
const badRequestMessage = "Неправильный формат данных или в них есть ошибка"

func (r *HttpRouter) Run() error {
	return r.App.Listen(":" + r.httpPort)
}

func (r *HttpRouter) Close() error {
	err := r.controller.Close()
	r.appLogger.Error("controller.Close failed: ", zap.Error(err))
	return r.App.Shutdown()
}

func (r *HttpRouter) Register(ctx *fiber.Ctx) error {
	request := &types.UserRequest{}
	err := ctx.BodyParser(request)
	if err != nil {
		r.appLogger.Error("ctx.BodyParser failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})

	}
	if request.UserName == "" || request.Password == "" {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}
	if request.FirstName == "" {
		request.FirstName = "Михал"
	}
	if request.LastName == "" {
		request.LastName = "Палыч"
	}
	err = r.controller.CreateNewUser(ctx.Context(), request)
	if errors.Is(err, database.ErrUserAlreadyExist) {
		r.appLogger.Error("controller.CreateNewUser failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Пользователь с таким ником уже существует"})
	}
	if err != nil {
		r.appLogger.Error("controller.CreateNewUser failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	ctx.Status(http.StatusCreated)
	return nil
}

func (r *HttpRouter) Login(ctx *fiber.Ctx) error {
	request := &types.UserRequest{}
	err := ctx.BodyParser(request)
	if err != nil {
		r.appLogger.Error("ctx.BodyParser failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})

	}
	if request.UserName == "" || request.Password == "" {
		return fiber.NewError(http.StatusBadRequest, badRequestMessage)
	}
	token, err := r.controller.AuthorizeUser(ctx.Context(), request)
	if errors.Is(err, database.ErrUserNotExist) {
		r.appLogger.Error("controller.AuthorizeUser failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Неправильный логин или пароль"})
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		r.appLogger.Error("controller.AuthorizeUser failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Неправильный логин или пароль"})
	}
	if err != nil {
		r.appLogger.Error("controller.AuthorizeUser failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	ctx.Status(http.StatusOK)
	return ctx.JSON(fiber.Map{"status": "success", "message": token})
}

func (r *HttpRouter) GetUserStatus(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}
	userId, err := strconv.Atoi(id)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}
	user, err := r.controller.GetUserStatus(ctx.Context(), userId)
	if errors.Is(err, database.ErrUserNotExist) {
		r.appLogger.Error("controller.GetUserStatus failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Пользователя с таким id несуществует"})
	}
	if err != nil {
		r.appLogger.Error("controller.GetUserStatus failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	ctx.Status(http.StatusOK)
	return ctx.JSON(user)
}

func (r *HttpRouter) GetLeaderBoard(ctx *fiber.Ctx) error {
	topUsers, err := r.controller.GetTopUsers(ctx.Context())
	if err != nil {
		r.appLogger.Error("controller.GetUserStatus failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	return ctx.JSON(topUsers)
}

func (r *HttpRouter) CompleteTask(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}
	userId, err := strconv.Atoi(id)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}
	taskRequest := &types.CompleteTaskRequest{}
	err = ctx.BodyParser(taskRequest)
	if err != nil {
		r.appLogger.Error("ctx.BodyParser failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})

	}
	if taskRequest.TaskId == 0 {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Необходим id задания"})
	}
	reward, err := r.controller.CompleteTask(ctx.Context(), userId, taskRequest.TaskId)
	if errors.Is(err, database.ErrUserNotExist) {
		r.appLogger.Error("controller.GetUserStatus failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Пользователя с таким id несуществует"})
	}
	if errors.Is(err, database.ErrTaskNotExist) {
		r.appLogger.Error("controller.GetUserStatus failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Задания с таким id несуществует"})
	}
	if errors.Is(err, database.ErrAlreadyCompletedTask) {
		r.appLogger.Error("controller.GetUserStatus failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Пользователь уже выполнил это задание"})
	}
	if err != nil {
		r.appLogger.Error("controller.GetUserStatus failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	ctx.Status(http.StatusOK)
	return ctx.JSON(fiber.Map{"status": "success", "reward": reward})
}

func (r *HttpRouter) Referrer(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}
	userId, err := strconv.Atoi(id)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}
	referrerRequest := &types.ReferrerRequest{}
	err = ctx.BodyParser(referrerRequest)
	if err != nil {
		r.appLogger.Error("ctx.BodyParser failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	if referrerRequest.ReferrerCode == "" {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Необходим реферальный код"})
	}
	err = r.controller.Referrer(ctx.Context(), userId, referrerRequest.ReferrerCode)
	if errors.Is(err, database.ErrUserNotExist) {
		r.appLogger.Error("controller.Referrer failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Такого реферального кода не существует"})
	}
	if err != nil {
		r.appLogger.Error("controller.GetUserStatus failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	ctx.Status(http.StatusOK)
	return nil
}

func (r *HttpRouter) CreateTask(ctx *fiber.Ctx) error {
	request := &types.Task{}
	err := ctx.BodyParser(request)
	if err != nil {
		r.appLogger.Error("ctx.BodyParser failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})

	}
	if request.Description == "" || request.Reward == 0 {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Заданию необходимо описание и награда"})
	}

	id, err := r.controller.CreateNewTask(ctx.Context(), request)
	if errors.Is(err, database.ErrTaskAlreadyExist) {
		r.appLogger.Error("controller.CreateNewTask failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Задание с таким описанием уже существует"})
	}
	if err != nil {
		r.appLogger.Error("controller.CreateNewTask failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	ctx.Status(http.StatusCreated)
	return ctx.JSON(fiber.Map{"status": "success", "id": id})
}

func (r *HttpRouter) UpdateTaskReward(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}
	taskId, err := strconv.Atoi(id)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}

	request := &types.Task{}
	err = ctx.BodyParser(request)
	if err != nil {
		r.appLogger.Error("ctx.BodyParser failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})

	}
	if request.Reward == 0 {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Заданию необходимо описание и награда"})
	}

	err = r.controller.UpdateTaskReward(ctx.Context(), taskId, request.Reward)
	if errors.Is(err, database.ErrTaskNotExist) {
		r.appLogger.Error("controller.UpdateTaskReward failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Задания с таким id несуществует"})
	}
	if err != nil {
		r.appLogger.Error("controller.UpdateTaskReward failed: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	ctx.Status(http.StatusOK)
	return nil
}

func (r *HttpRouter) GetAllTasks(ctx *fiber.Ctx) error {
	tasks, err := r.controller.GetAllTasks(ctx.Context())
	if err != nil {
		r.appLogger.Error("controller.GetAllTasks: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	return ctx.JSON(tasks)
}

func (r *HttpRouter) GetTask(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}
	taskId, err := strconv.Atoi(id)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": badRequestMessage})
	}
	task, err := r.controller.GetTask(ctx.Context(), taskId)
	if errors.Is(err, database.ErrTaskNotExist) {
		r.appLogger.Error("controller.GetTask failed: ", zap.Error(err))
		ctx.Status(http.StatusBadRequest)
		return ctx.JSON(fiber.Map{"status": "error", "message": "Задания с таким id несуществует"})
	}
	if err != nil {
		r.appLogger.Error("controller.GetTask: ", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"status": "error", "message": internalServerErrorMessage})
	}
	return ctx.JSON(task)
}

func CreateRouter(c controller, cfg *config.Config, logger *zap.Logger) *HttpRouter {
	appLogger := logger.Named("app")
	app := fiber.New()
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))

	r := &HttpRouter{controller: c, App: app, appLogger: appLogger, httpPort: cfg.HttpPort}
	api := r.Group("/api/v1")
	api.Post("/register", r.Register)
	api.Post("/login", r.Login)
	users := api.Group("/users", middleware.Protected([]byte(cfg.JWTSecret)))
	users.Get("/:id/status", r.GetUserStatus)
	users.Get("/leaderboard", r.GetLeaderBoard)
	users.Post("/:id/task/complete", r.CompleteTask)
	users.Post("/:id/referrer", r.Referrer)

	tasks := api.Group("/tasks", middleware.Protected([]byte(cfg.JWTSecret)))
	tasks.Get("/all", r.GetAllTasks)
	tasks.Get("/:id", r.GetTask)
	tasks.Post("/create", r.CreateTask)
	tasks.Post("/:id/updateReward", r.UpdateTaskReward)
	return r
}
