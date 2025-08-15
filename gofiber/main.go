package main

import (
	"log"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
)

var (
	largeText = strings.Repeat("The quick brown fox jumps over the lazy dog. ", 512)
	reSeg     = regexp.MustCompile(`^[a-z0-9-]{1,16}$`)
)

func requestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = "generated"
		}
		c.Set("X-Request-ID", id)
		c.Locals("rid", id)
		return c.Next()
	}
}

// A minimal Fiber v3 server: GET /ping -> "pong"
func main() {
	app := fiber.New()

	// 1) Simple ping
	app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })

	// 2) Middleware and 3) Context
	mw := app.Group("/mw", requestID())
	mw.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })
	mw.Get("/ctx", func(c fiber.Ctx) error {
		if v := c.Locals("rid"); v != nil {
			return c.SendString(v.(string))
		}
		return c.SendString("no-id")
	})

	// 4) JSON decode
	type userIn struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
		Ok    bool   `json:"ok"`
		Items []int  `json:"items"`
	}
	app.Post("/json", func(c fiber.Ctx) error {
		var in userIn
		if err := c.Bind().JSON(&in); err != nil {
			return c.Status(400).SendString("bad json")
		}
		return c.SendString("ok")
	})

	// 5) Nested groups (basic)
	api := app.Group("/api")
	v1 := api.Group("/v1")
	grp := v1.Group("/group")
	grp.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })

	// Param, Wildcard, Regex
	app.Get("/param/:id", func(c fiber.Ctx) error { return c.SendString(c.Params("id")) })
	app.Get("/wild/*", func(c fiber.Ctx) error { return c.SendString(c.Params("*")) })

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
	g10.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })

	// 10 middleware chain
	var chain []fiber.Handler
	for i := 0; i < 10; i++ {
		chain = append(chain, func(c fiber.Ctx) error { return c.Next() })
	}
	cmw := app.Group("/mw10", chain...)
	cmw.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })

	log.Fatal(app.Listen(":18082"))
}
