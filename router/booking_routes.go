package router

import (
	"go-starter/internal/controller"

	"github.com/gin-gonic/gin"
)

// registerBookingRoutes wires booking creation and per-user listing.
func registerBookingRoutes(group *gin.RouterGroup, srv *controller.Server) {
	group.POST("/bookings", srv.CreateBooking)
	group.GET("/users/:user_id/bookings", srv.ListUserBookings)
}
