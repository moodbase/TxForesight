package main

import (
	"github.com/moodbase/TxForesight/server"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	s := server.New(":8080")
	err := s.Start()
	if err != nil {
		slog.Error("failed to start server", "err", err)
		return
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.Stop()
}
