package proxy

import (
	"net/http"
	"time"

	ginzap "github.com/gin-contrib/zap"
	gin "github.com/gin-gonic/gin"
	zap "go.uber.org/zap"
)

type ServerOptions struct {
	ListenAddress string
}

// Server creates http proxy server
func (proxy Proxy) Server(options ServerOptions) *http.Server {
	router := gin.New()

	logger, _ := zap.NewProduction()
	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))

	router.GET("/:scope/:name", proxy.getPackageHandler)
	router.GET("/:scope", proxy.getPackageHandler)
	router.NoRoute(proxy.noRouteHandler)

	return &http.Server{
		Handler: router,
		Addr:    options.ListenAddress,
	}
}

func (proxy Proxy) getPackageHandler(c *gin.Context) {
	var name string
	if c.Param("name") != "" {
		name = c.Param("scope") + "/" + c.Param("name")
	} else {
		name = c.Param("scope")
	}

	pkg, err := proxy.GetMetadata(name, c.Request.URL.Path, c.Request)

	if err != nil {
		c.AbortWithError(500, err)
	} else {
		// c.Header("Content-Encoding", "gzip")
		c.Data(200, "application/json", pkg)
	}
}

func (proxy Proxy) noRouteHandler(c *gin.Context) {
	if c.Request.URL.Path == "/" {
		err := proxy.Database.Health()

		if err != nil {
			c.AbortWithStatusJSON(503, err)
		} else {
			c.AbortWithStatusJSON(200, gin.H{"ok": true})
		}
	} else {
		options, err := proxy.GetOptions()

		if err != nil {
			c.AbortWithStatusJSON(500, err)
		} else {
			c.Redirect(http.StatusTemporaryRedirect, options.UpstreamAddress+c.Request.URL.Path)
		}
	}
}
