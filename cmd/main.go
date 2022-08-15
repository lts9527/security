package main

import (
	"security/config"
	"security/internal/handler"
	"security/log"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Error("panic: %#v\n", err)
		}
	}()
	config.InitConfig()
	handler.Handier()
}
