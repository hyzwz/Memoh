package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type PingHandler struct{}

func NewPingHandler() *PingHandler {
	return &PingHandler{}
}

func (h *PingHandler) Register(e *echo.Echo) {
	e.GET("/ping", h.Ping)
}

func (h *PingHandler) Ping(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}
