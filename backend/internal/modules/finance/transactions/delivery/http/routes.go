package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/finance/transactions", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Route("/{transactionID}", func(r chi.Router) {
			r.Get("/", h.GetByID)
			r.Put("/", h.Update)
			r.Delete("/", h.Delete)
			r.Get("/attachments", h.ListAttachments)
			r.Post("/attachments", h.UploadAttachment)
			r.Get("/attachments/{attachmentID}/download", h.DownloadAttachment)
			r.Patch("/attachments/{attachmentID}/scan-status", h.SetAttachmentScanStatus)
		})
		r.Get("/by-savings/{savingsID}", h.ListBySavingsID)
	})
}

func RegisterHealth(r chi.Router) {
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}
