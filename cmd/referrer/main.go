package main

import (
	"github.com/SakuraBurst/denet/internal/referrer"
	"github.com/SakuraBurst/denet/internal/referrer/config"
)

func main() {
	cfg := config.MustLoadPath("./config/config.yaml")
	a := referrer.NewApp(cfg)
	a.Run()
}
