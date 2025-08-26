package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
)

var (
	largeText = strings.Repeat("The quick brown fox jumps over the lazy dog. ", 512)
	reSeg     = regexp.MustCompile(`^[a-z0-9-]{1,16}$`)
)

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = "generated"
		}
		w.Header().Set("X-Request-ID", id)
		ctx := r.Context()
		ctx = context.WithValue(ctx, "rid", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// A minimal Chi server: GET /ping -> "pong"
func main() {
	r := chi.NewRouter()

	// 1) Simple ping
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// 2) Middleware and 3) Context
	r.Route("/mw", func(r chi.Router) {
		r.Use(requestIDMiddleware)
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		})
		r.Get("/ctx", func(w http.ResponseWriter, r *http.Request) {
			if v := r.Context().Value("rid"); v != nil {
				w.Write([]byte(v.(string)))
			} else {
				w.Write([]byte("no-id"))
			}
		})
	})

	// Context route for benchmark
	r.Get("/context", func(w http.ResponseWriter, r *http.Request) {
		// Set a value in context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "test-key", "test-value")
		r = r.WithContext(ctx)

		// Read the value back
		if v := r.Context().Value("test-key"); v != nil {
			w.Write([]byte(v.(string)))
		} else {
			w.Write([]byte("context-ok"))
		}
	})

	// 4) JSON decode
	type userIn struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
		Ok    bool   `json:"ok"`
		Items []int  `json:"items"`
	}
	r.Post("/json", func(w http.ResponseWriter, r *http.Request) {
		var in userIn
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		w.Write([]byte("ok"))
	})

	// 5) Nested groups (basic)
	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Route("/group", func(r chi.Router) {
				r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("pong"))
				})
			})
		})
	})

	// Param, Wildcard, Regex
	r.Get("/param/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(chi.URLParam(r, "id")))
	})
	r.Get("/wild/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(chi.URLParam(r, "*")))
	})
	r.Get("/wildcard/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(chi.URLParam(r, "*")))
	})

	// 10 nested groups
	r.Route("/g1", func(r chi.Router) {
		r.Route("/g2", func(r chi.Router) {
			r.Route("/g3", func(r chi.Router) {
				r.Route("/g4", func(r chi.Router) {
					r.Route("/g5", func(r chi.Router) {
						r.Route("/g6", func(r chi.Router) {
							r.Route("/g7", func(r chi.Router) {
								r.Route("/g8", func(r chi.Router) {
									r.Route("/g9", func(r chi.Router) {
										r.Route("/g10", func(r chi.Router) {
											r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
												w.Write([]byte("pong"))
											})
										})
									})
								})
							})
						})
					})
				})
			})
		})
	})

	// 10 middleware chain
	r.Route("/mw10", func(r chi.Router) {
		// Add 10 middleware functions
		for i := 0; i < 10; i++ {
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					next.ServeHTTP(w, r)
				})
			})
		}
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		})
	})

	log.Fatal(http.ListenAndServe(":17784", r))
}
