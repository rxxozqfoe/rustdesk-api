package web

import (
	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
)

type Index struct {
	HD *deps.HandlerDeps
}

func (i *Index) Index(c *gin.Context) {
	c.Redirect(302, "/_admin/")
}
