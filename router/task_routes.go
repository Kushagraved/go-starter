package router

import (
	"go-starter/internal/controller"

	"github.com/gin-gonic/gin"
)

func registerTaskRoutes(rg *gin.RouterGroup, srv *controller.Server) {
	tasks := rg.Group("/tasks")
	{
		tasks.POST("", srv.CreateTask)
		tasks.GET("", srv.ListTasks)
		tasks.GET("/:id", srv.GetTask)
		tasks.PATCH("/:id", srv.UpdateTask)
		tasks.DELETE("/:id", srv.DeleteTask)
	}
}
