package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

type PageInfo struct {
	Page     int `form:"page" json:"page" binding:"required"`
	PageSize int `form:"pageSize" json:"pageSize" binding:"required"`
}

func (s *Server) routeETH(tag ChainTag) {
	// GET /{tag}
	g := s.r.Group(string(tag))
	g.GET("/tx-pool", func(ctx *gin.Context) {
		var pageInfo PageInfo
		err := ctx.ShouldBind(&pageInfo)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			slog.Error("failed to bind json", "err", err)
			return
		}
		pool := s.ethPools[tag]
		selected, total := pool.All(pageInfo.Page, pageInfo.PageSize)
		ctx.JSON(http.StatusOK, gin.H{
			"selected": selected,
			"total":    total,
		})
	})
	g.GET("/tx-pool/:hash", func(ctx *gin.Context) {
		hashStr := ctx.Param("hash")
		pool := s.ethPools[tag]
		var hash common.Hash
		err := hash.UnmarshalText([]byte(hashStr))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		tx, _ := pool.Get(hash)
		ctx.JSON(http.StatusOK, gin.H{
			hashStr: tx,
		})
	})
	g.GET("/chain-config", func(ctx *gin.Context) {
		config := s.ethPoolServers[tag].ChainConfig()
		ctx.Data(http.StatusOK, "application/json", config)
	})
}

func (s *Server) httpListen() {
	err := s.httpListener.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start http server", "err", err)
		return
	}
}

func (s *Server) httpShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.httpListener.Shutdown(ctx); err != nil {
		slog.Error("failed to shutdown http server", "err", err)
	}
	slog.Info("http server stopped")
}
