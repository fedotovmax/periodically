package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/fedotovmax/periodically"
)

func main() {

	log := slog.Default()

	periodicallyManager := periodically.NewManager(log)

	periodicallyManager.Every(time.Second*5, func(ctx context.Context) {
		log.Info("выполняюсь раз в 5 сек.")
	})

	periodicallyManager.Every(time.Second*10, func(ctx context.Context) {
		log.Info("выполняюсь раз в 10 сек.")
	})

	periodicallyManager.Start()

	<-time.After(time.Minute)

	shutdown, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := periodicallyManager.Stop(shutdown)

	if err != nil {
		log.Error(err.Error())
	}

	log.Info("app stopped")

}
