package main

import (
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
)

var (
	largeText = strings.Repeat("The quick brown fox jumps over the lazy dog. ", 512)
	reSeg     = regexp.MustCompile(`^[a-z0-9-]{1,16}$`)
)

func requestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get("X-Request-ID")
			if id == "" {
				id = "generated"
			}
			c.Response().Header().Set("X-Request-ID", id)
			c.Set("rid", id)
			return next(c)
		}
	}
}

// A minimal Echo server: GET /ping -> "pong"
func main() {
	e := echo.New()

	// 1) Simple ping
	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	// 2) Middleware and 3) Context
	mw := e.Group("/mw", requestIDMiddleware())
	mw.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})
	mw.GET("/ctx", func(c echo.Context) error {
		if v, ok := c.Get("rid").(string); ok {
			return c.String(http.StatusOK, v)
		}
		return c.String(http.StatusOK, "no-id")
	})

	// Context route for benchmark
	e.GET("/context", func(c echo.Context) error {
		// Set a value in context
		c.Set("test-key", "test-value")

		// Read the value back
		if v, ok := c.Get("test-key").(string); ok {
			return c.String(http.StatusOK, v)
		}
		return c.String(http.StatusOK, "context-ok")
	})

	// 4) JSON decode
	type userIn struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
		Ok    bool   `json:"ok"`
		Items []int  `json:"items"`
	}
	e.POST("/json", func(c echo.Context) error {
		var in userIn
		if err := c.Bind(&in); err != nil {
			return c.String(http.StatusBadRequest, "bad json")
		}
		return c.String(http.StatusOK, "ok")
	})

	// 5) Nested groups (basic)
	api := e.Group("/api")
	v1 := api.Group("/v1")
	grp := v1.Group("/group")
	grp.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	// Param, Wildcard, Regex
	e.GET("/param/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, c.Param("id"))
	})
	e.GET("/wild/*", func(c echo.Context) error {
		return c.String(http.StatusOK, c.Param("*"))
	})
	e.GET("/wildcard/*", func(c echo.Context) error {
		return c.String(http.StatusOK, c.Param("*"))
	})

	// 10 nested groups
	g1 := e.Group("/g1")
	g2 := g1.Group("/g2")
	g3 := g2.Group("/g3")
	g4 := g3.Group("/g4")
	g5 := g4.Group("/g5")
	g6 := g5.Group("/g6")
	g7 := g6.Group("/g7")
	g8 := g7.Group("/g8")
	g9 := g8.Group("/g9")
	g10 := g9.Group("/g10")
	g10.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	// 10 middleware chain
	var chain []echo.MiddlewareFunc
	for i := 0; i < 10; i++ {
		chain = append(chain, func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				return next(c)
			}
		})
	}
	cmw := e.Group("/mw10", chain...)
	cmw.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "17783"
	}
	log.Fatal(e.Start(":" + port))
}
