package http

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/settings/whatsapp", func(r chi.Router) {
		r.Post("/otp", h.GetOTP)
		r.Post("/connect", h.Connect)
		r.Get("/status", h.GetStatus)
		r.Delete("/disconnect", h.Disconnect)
		r.Post("/chat", h.WebChat)
	})
}
