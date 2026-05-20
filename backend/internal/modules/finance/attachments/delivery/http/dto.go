package http

type AttachmentResponse struct {
	ID               string `json:"id"`
	OwnerType        string `json:"owner_type"`
	OwnerID          string `json:"owner_id"`
	OriginalFilename string `json:"original_filename"`
	MimeType         string `json:"mime_type"`
	ScanStatus       string `json:"scan_status"`
	SizeBytes        int64  `json:"size_bytes"`
	CreatedAt        string `json:"created_at"`
}

