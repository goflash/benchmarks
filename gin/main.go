package main

import (
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	largeText = strings.Repeat("The quick brown fox jumps over the lazy dog. ", 512)
	reSeg     = regexp.MustCompile(`^[a-z0-9-]{1,16}$`)
)

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Request.Header.Get("X-Request-ID")
		if id == "" {
			id = "generated"
		}
		c.Writer.Header().Set("X-Request-ID", id)
		c.Set("rid", id)
		c.Next()
	}
}

// A minimal Gin server: GET /ping -> "pong"
func main() {
	r := gin.New()
	r.HandleMethodNotAllowed = true

	// 1) Simple ping
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// 2) Middleware and 3) Context
	mw := r.Group("/mw", requestIDMiddleware())
	mw.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	mw.GET("/ctx", func(c *gin.Context) {
		if v, ok := c.Get("rid"); ok {
			c.String(http.StatusOK, v.(string))
			return
		}
		c.String(http.StatusOK, "no-id")
	})

	// 4) JSON decode
	type userIn struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
		Ok    bool   `json:"ok"`
		Items []int  `json:"items"`
	}
	r.POST("/json", func(c *gin.Context) {
		var in userIn
		if err := c.ShouldBindJSON(&in); err != nil {
			c.String(http.StatusBadRequest, "bad json")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	// 5) Nested groups (basic)
	api := r.Group("/api")
	v1 := api.Group("/v1")
	grp := v1.Group("/group")
	grp.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	// Param, Wildcard, Regex
	r.GET("/param/:id", func(c *gin.Context) { c.String(http.StatusOK, c.Param("id")) })
	r.GET("/wild/*path", func(c *gin.Context) { c.String(http.StatusOK, c.Param("path")) })

	// 10 nested groups
	g1 := r.Group("/g1")
	g2 := g1.Group("/g2")
	g3 := g2.Group("/g3")
	g4 := g3.Group("/g4")
	g5 := g4.Group("/g5")
	g6 := g5.Group("/g6")
	g7 := g6.Group("/g7")
	g8 := g7.Group("/g8")
	g9 := g8.Group("/g9")
	g10 := g9.Group("/g10")
	g10.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	// 10 middleware chain
	var chain []gin.HandlerFunc
	for i := 0; i < 10; i++ {
		chain = append(chain, func(c *gin.Context) { c.Next() })
	}
	cmw := r.Group("/mw10", chain...)
	cmw.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	log.Fatal(http.ListenAndServe(":18081", r))
}
