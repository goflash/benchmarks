package main

import (
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/goflash/flash/v2"
	"github.com/goflash/flash/v2/middleware"
)

var (
	largeText = strings.Repeat("The quick brown fox jumps over the lazy dog. ", 512)
	reSeg     = regexp.MustCompile(`^[a-z0-9-]{1,16}$`)
)

// GoFlash server implementing all required endpoints
func main() {
	app := flash.New()

	// 1) Simple ping
	app.GET("/ping", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})

	// 2) Middleware and 3) Context
	mw := app.Group("/mw", middleware.RequestID())
	mw.GET("/ping", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})
	mw.GET("/ctx", func(c flash.Ctx) error {
		if id, ok := middleware.RequestIDFromContext(c.Context()); ok {
			return c.String(http.StatusOK, id)
		}
		return c.String(http.StatusOK, "no-id")
	})

	// Context route for benchmark
	app.GET("/context", func(c flash.Ctx) error {
		// Set a value in context
		c.Set("test-key", "test-value")

		// Read the value back
		if v := c.Get("test-key"); v != nil {
			return c.String(http.StatusOK, v.(string))
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
	app.POST("/json", func(c flash.Ctx) error {
		var in userIn
		if err := c.BindJSON(&in); err != nil {
			return c.String(http.StatusBadRequest, "bad json")
		}
		return c.String(http.StatusOK, "ok")
	})

	// 5) Nested groups (basic)
	api := app.Group("/api")
	v1 := api.Group("/v1")
	grp := v1.Group("/group")
	grp.GET("/ping", func(c flash.Ctx) error { return c.String(http.StatusOK, "pong") })

	// Param, Wildcard, Regex examples
	app.GET("/param/:id", func(c flash.Ctx) error {
		return c.String(http.StatusOK, c.Param("id"))
	})
	app.GET("/wild/*path", func(c flash.Ctx) error {
		return c.String(http.StatusOK, c.Param("path"))
	})
	app.GET("/wildcard/*path", func(c flash.Ctx) error {
		return c.String(http.StatusOK, c.Param("path"))
	})

	// 10 nested groups
	g1 := app.Group("/g1")
	g2 := g1.Group("/g2")
	g3 := g2.Group("/g3")
	g4 := g3.Group("/g4")
	g5 := g4.Group("/g5")
	g6 := g5.Group("/g6")
	g7 := g6.Group("/g7")
	g8 := g7.Group("/g8")
	g9 := g8.Group("/g9")
	g10 := g9.Group("/g10")
	g10.GET("/ping", func(c flash.Ctx) error { return c.String(http.StatusOK, "pong") })

	// Chain of 10 middlewares
	var chain []flash.Middleware
	for i := 0; i < 10; i++ {
		chain = append(chain, func(next flash.Handler) flash.Handler {
			return func(c flash.Ctx) error { return next(c) }
		})
	}
	cmw := app.Group("/mw10", chain...)
	cmw.GET("/ping", func(c flash.Ctx) error { return c.String(http.StatusOK, "pong") })

	port := os.Getenv("PORT")
	if port == "" {
		port = "17780"
	}

	log.Printf("Starting GoFlash server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, app))
}
