package web

import (
	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
)

type Index struct {
	HD *deps.HandlerDeps
}

func (i *Index) Index(c *gin.Context) {
	// The API is back-end only now; the admin UI is served by the separate
	// front-end deployment. Return a small JSON so "/" is not a dead redirect.
	c.JSON(200, gin.H{"code": 0, "msg": "rustdesk-api"})
}
