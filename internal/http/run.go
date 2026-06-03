//go:build !windows

package http

import (
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
)

func Run(g *gin.Engine, addr string) {
	if err := endless.ListenAndServe(addr, g); err != nil {
		panic(err)
	}
}
