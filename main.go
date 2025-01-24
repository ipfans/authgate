package main

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/ipfans/authgate/config"
	"github.com/ipfans/authgate/routers"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	h := server.Default()
	routers.RegisterRoutes(h, cfg.Routes)
	h.Spin()
}
