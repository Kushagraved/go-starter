package router

import (
	"go-starter/internal/controller"

	"github.com/gin-gonic/gin"
)

// registerOnboardingRoutes wires the setup endpoints: branches, their fleet,
// pricing, and users.
func registerOnboardingRoutes(group *gin.RouterGroup, srv *controller.Server) {
	branches := group.Group("/branches")
	{
		branches.POST("", srv.CreateBranch)
		branches.POST("/:branch_id/cars", srv.CreateCar)
		branches.PUT("/:branch_id/prices", srv.SetPrice)
	}

	group.POST("/users", srv.CreateUser)
}
