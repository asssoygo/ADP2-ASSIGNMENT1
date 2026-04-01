package http

import "github.com/gin-gonic/gin"

func NewRouter(handler *OrderHandler) *gin.Engine {
	r := gin.Default()

	r.POST("/orders", handler.CreateOrder)
	r.GET("/orders/:id", handler.GetOrder)
	r.PATCH("/orders/:id/cancel", handler.CancelOrder)

	return r
}
