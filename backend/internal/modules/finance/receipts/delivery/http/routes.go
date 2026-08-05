package http

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/finance/settings/receipt-scan", func(r chi.Router) {
		r.Get("/providers", h.ListProviders)
		r.Put("/providers", h.UpdateProviders)
		r.Post("/providers/test", h.TestProviderConnection)
	})
	r.Route("/finance/receipt-scan", func(r chi.Router) {
		r.Get("/status", h.GetStatus)
		r.Get("/history", h.ListHistory)
		r.Delete("/history", h.ClearHistory)
		r.Delete("/history/{id}", h.DeleteHistoryItem)
		r.Get("/history/{id}/image", h.GetReceiptImage)
		r.Post("/scan", h.ScanReceipt)
	})
}
