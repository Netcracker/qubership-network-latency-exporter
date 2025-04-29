package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type shutdown struct {
	srv     *http.Server
	logger  *slog.Logger
	ctx     context.Context
	timeout time.Duration
}

func (s *shutdown) listen() {
	s.logger.Debug("Start listening for interruption signal")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	s.do()
}

func (s *shutdown) do() {
	s.logger.Info("Try to shut down server gracefully", "timeout", s.timeout)
	shtdwnCtx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()
	if err := s.srv.Shutdown(shtdwnCtx); err != nil {
		s.logger.Info("Failed to shut down server gracefully", "timeout", s.timeout, "err", err)
		s.logger.Info("Force closing server", "err", s.srv.Close())
	}
}
