package main

import (
	"github.com/SakuraBurst/denet/internal/referrer"
	"github.com/SakuraBurst/denet/internal/referrer/config"
)

func main() {
	// "./config/config.yaml"
	cfg := config.MustLoad()
	a := referrer.NewApp(cfg)
	a.Run()
}
