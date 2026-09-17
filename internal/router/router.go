package router

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/router/debug"
)

func StartRouter() {
	r := newRouter()

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func newRouter() *gin.Engine {
	r := gin.Default()

	debug.HealthRoutes(r)

	return r
}
