package referrer

import (
	"os"
	"os/signal"

	"github.com/SakuraBurst/denet/internal/pkg/logger"
	"github.com/SakuraBurst/denet/internal/referrer/config"
	"github.com/SakuraBurst/denet/internal/referrer/controller"
	"github.com/SakuraBurst/denet/internal/referrer/database"
	"github.com/SakuraBurst/denet/internal/referrer/router"
	"go.uber.org/zap"
)

type App struct {
	router *router.HttpRouter
	logger *zap.Logger
}

func (a *App) Run() error {
	sisChan := make(chan os.Signal, 1)
	go func() {
		if err := a.router.Run(); err != nil {
			a.logger.Error("router.Run failed: ", zap.Error(err))
			sisChan <- os.Interrupt
		}
	}()
	return a.gracefulShutdown(sisChan)
}

func (a *App) gracefulShutdown(sisChan chan os.Signal) error {
	signal.Notify(sisChan, os.Interrupt)
	<-sisChan
	err := a.router.Close()
	if err != nil {
		a.logger.Error("router.Close failed: ", zap.Error(err))
	}
	return a.logger.Sync()
}

func NewApp(cfg *config.Config) *App {
	log, err := logger.InitLogger()
	if err != nil {
		panic(err)
	}
	db, err := database.NewDB(cfg, log)
	if err != nil {
		panic(err)
	}
	c := controller.NewController(cfg, db, db, db, func() error {
		db.Conn.Close()
		return nil
	})
	r := router.CreateRouter(c, cfg, log)
	return &App{
		router: r,
		logger: log,
	}
}
