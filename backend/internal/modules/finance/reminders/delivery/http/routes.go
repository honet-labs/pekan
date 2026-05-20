package http

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/finance/reminders", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/due", h.ListDue)
		r.Route("/{reminderID}", func(r chi.Router) {
			r.Get("/", h.GetByID)
			r.Put("/", h.Update)
			r.Delete("/", h.Delete)
			r.Post("/status", h.MarkStatus)
			r.Post("/payments", h.AddPayment)
			r.Get("/payments", h.GetPaymentHistory)
			r.Route("/payments/{paymentID}", func(r chi.Router) {
				r.Put("/", h.UpdatePayment)
				r.Delete("/", h.DeletePayment)
				r.Get("/proof", h.GetProofImage)
			})
		})
	})
}

