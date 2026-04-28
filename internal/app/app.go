package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"room-booking-service/config"
	"room-booking-service/internal/controller/http"
	repo "room-booking-service/internal/repo/postgres"
	usecase "room-booking-service/internal/usecase/booking"
	"room-booking-service/pkg/httpserver"
	"room-booking-service/pkg/logger"
	"room-booking-service/pkg/postgres"
	"syscall"
)

func Run(cfg *config.Config) {
	l := logger.New(cfg.Log.Level)
	appCtx, cancel := context.WithCancel(context.Background())

	// Repository
	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		err := errors.Join(errors.New("app - Run - postgres.New"), err)
		l.Fatal(err)
	}
	defer pg.Close()
	repo := repo.NewRepo(pg)
	// Service
	service := usecase.New(repo, l)
	go service.AddSlots(appCtx)

	httpServer := httpserver.New(httpserver.Port(cfg.HTTP.Port), httpserver.Prefork(cfg.HTTP.UsePreforkMode))
	http.NewRouter(httpServer.App, cfg, service, l)

	httpServer.Start()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: %s", s.String())
	case err = <-httpServer.Notify():
		l.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
	}
	err = httpServer.Shutdown()
	cancel()
	if err != nil {
		l.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}

}
