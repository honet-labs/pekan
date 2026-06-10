package usecase

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"pekan/backend/internal/modules/finance/whatsapp/domain"
)

type Authorizer interface {
	EnsurePermission(ctx context.Context, permission string) error
	EnsureModule(ctx context.Context, module string) error
	EnsureFeature(ctx context.Context, feature string) error
}

type SettingsService interface {
	GetGlobalSettingRaw(ctx context.Context, key string) (string, error)
}

type pendingTransaction struct {
	Amount       int64
	Type         string
	Description  string
	CategoryName string
	Date         string
	Items        []domain.ChatItem
	ExpiresAt    time.Time
}

type Service struct {
	repo         domain.Repository
	authz        Authorizer
	settings     SettingsService
	pendingScans map[string]pendingTransaction
	pendingMu    sync.RWMutex
}

func NewService(repo domain.Repository, authz Authorizer, settings SettingsService) *Service {
	s := &Service{
		repo:         repo,
		authz:        authz,
		settings:     settings,
		pendingScans: make(map[string]pendingTransaction),
	}
	return s
}

// GenerateOTPCode creates a short 6-character uppercase alphanumeric string
func generateOTPCode() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "WA0000" // Fallback (should ideally never happen)
	}
	return "WA-" + strings.ToUpper(hex.EncodeToString(b))
}

type GenerateOTPInput struct {
	TenantID string
	UserID   string
}

func (s *Service) GenerateOTP(ctx context.Context, in GenerateOTPInput) (string, error) {
	// First delete expired tokens to keep the table clean
	_ = s.repo.DeleteExpiredTokens(ctx)

	tokenStr := generateOTPCode()
	now := time.Now().UTC()
	
	err := s.repo.CreateOTPToken(ctx, domain.OTPToken{
		Token:     tokenStr,
		TenantID:  in.TenantID,
		UserID:    in.UserID,
		ExpiresAt: now.Add(10 * time.Minute), // Valid for 10 minutes
		CreatedAt: now,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	return tokenStr, nil
}

func (s *Service) GetConnectionStatus(ctx context.Context, tenantID, userID string) (domain.Session, error) {
	session, err := s.repo.GetSessionByUser(ctx, tenantID, userID)
	if err != nil {
		return domain.Session{}, err
	}
	return session, nil
}

func (s *Service) Disconnect(ctx context.Context, tenantID, userID string) error {
	return s.repo.DeleteSessionByUser(ctx, tenantID, userID)
}

func cleanPhoneNumber(phone string) string {
	phone = strings.TrimSpace(phone)
	if strings.Contains(phone, "@") {
		phone = strings.Split(phone, "@")[0]
	}
	var sb strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	digits := sb.String()
	if strings.HasPrefix(digits, "620") {
		digits = "0" + strings.TrimPrefix(digits, "620")
	} else if strings.HasPrefix(digits, "62") {
		digits = "0" + strings.TrimPrefix(digits, "62")
	}
	if strings.HasPrefix(digits, "8") {
		digits = "0" + digits
	}
	return digits
}

func (s *Service) ConnectDirect(ctx context.Context, tenantID, userID, phoneNumber string) error {
	cleanPhone := cleanPhoneNumber(phoneNumber)
	if cleanPhone == "" {
		return fmt.Errorf("phone number cannot be empty")
	}

	// Ensure phone is not already used
	existing, err := s.repo.GetSessionByPhone(ctx, cleanPhone)
	if err == nil {
		if existing.TenantID == tenantID && existing.UserID == userID {
			return nil // Already connected exactly as requested
		}
		return domain.ErrAlreadyConnected
	}

	// Delete any existing session for this user first
	_ = s.repo.DeleteSessionByUser(ctx, tenantID, userID)

	now := time.Now().UTC()
	err = s.repo.CreateSession(ctx, domain.Session{
		PhoneNumber: cleanPhone,
		TenantID:    tenantID,
		UserID:      userID,
		LastActive:  &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (s *Service) resolveJIDViaWAHA(ctx context.Context, phone string) (string, error) {
	configJSON, err := s.settings.GetGlobalSettingRaw(ctx, "notification_wa_waha")
	if err != nil || configJSON == "" {
		return "", fmt.Errorf("konfigurasi WAHA belum diatur")
	}
	var cfg struct {
		ApiUrl  string `json:"apiUrl"`
		ApiKey  string `json:"apiKey"`
		Session string `json:"session"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil || cfg.ApiUrl == "" {
		return "", fmt.Errorf("konfigurasi WAHA tidak valid")
	}

	session := cfg.Session
	if session == "" {
		session = "default"
	}

	// Clean the phone number to digits only (e.g. 62859...)
	var cleanPhone string
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			cleanPhone += string(char)
		}
	}
	// If it starts with 08, replace with 628
	if strings.HasPrefix(cleanPhone, "08") {
		cleanPhone = "62" + strings.TrimPrefix(cleanPhone, "0")
	}

	baseApiUrl := cfg.ApiUrl
	if idx := strings.Index(baseApiUrl, "/api/"); idx != -1 {
		baseApiUrl = baseApiUrl[:idx]
	}
	baseApiUrl = strings.TrimSuffix(baseApiUrl, "/")

	// Attempt 1: GET /api/contacts/check-exists?phone={phone}&session={session}
	checkUrl := fmt.Sprintf("%s/api/contacts/check-exists?phone=%s&session=%s", baseApiUrl, cleanPhone, session)
	req, err := http.NewRequestWithContext(ctx, "GET", checkUrl, nil)
	if err == nil {
		if cfg.ApiKey != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)
			req.Header.Set("X-Api-Key", cfg.ApiKey)
		}
		resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var respPayload struct {
					NumberExists bool   `json:"numberExists"`
					ChatID       string `json:"chatId"`
				}
				if json.NewDecoder(resp.Body).Decode(&respPayload) == nil && respPayload.NumberExists && respPayload.ChatID != "" {
					return respPayload.ChatID, nil
				}
			}
		}
	}

	// Attempt 2 Fallback: GET /api/{session}/lids/pn/{phoneNumber}
	lidUrl := fmt.Sprintf("%s/api/%s/lids/pn/%s", baseApiUrl, session, cleanPhone)
	reqLid, err := http.NewRequestWithContext(ctx, "GET", lidUrl, nil)
	if err == nil {
		if cfg.ApiKey != "" {
			reqLid.Header.Set("Authorization", "Bearer "+cfg.ApiKey)
			reqLid.Header.Set("X-Api-Key", cfg.ApiKey)
		}
		respLid, err := (&http.Client{Timeout: 3 * time.Second}).Do(reqLid)
		if err == nil {
			defer respLid.Body.Close()
			if respLid.StatusCode == http.StatusOK {
				var respPayload struct {
					LID string `json:"lid"`
					PN  string `json:"pn"`
				}
				if json.NewDecoder(respLid.Body).Decode(&respPayload) == nil && respPayload.LID != "" {
					return respPayload.LID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("gagal me-resolve JID dari WAHA menggunakan kedua endpoint")
}

func (s *Service) ProcessLogin(ctx context.Context, phoneNumber, code string) error {
	// 1. Validate the code
	token, err := s.repo.GetOTPToken(ctx, code)
	if err != nil {
		return err // ErrTokenNotFound
	}
	
	if time.Now().UTC().After(token.ExpiresAt) {
		_ = s.repo.DeleteOTPToken(ctx, code)
		return domain.ErrTokenNotFound
	}

	// 1.5. Validate phone number matches the user's registered phone
	expectedPhone, err := s.repo.GetUserPhone(ctx, token.UserID)
	if err != nil {
		return fmt.Errorf("gagal memverifikasi profil pengguna: %w", err)
	}
	if expectedPhone == "" {
		return fmt.Errorf("anda belum mengatur nomor handphone di profil Anda, harap isi Nomor HP di halaman WebUI Profil Anda")
	}

	// Normalize both phone numbers for comparison
	cleanExpected := cleanPhoneNumber(expectedPhone)
	cleanReceived := cleanPhoneNumber(phoneNumber)

	match := false
	if cleanExpected == cleanReceived {
		match = true
	} else {
		resolvedJID, rErr := s.resolveJIDViaWAHA(ctx, expectedPhone)
		if rErr == nil && resolvedJID != "" {
			cleanResolved := cleanPhoneNumber(resolvedJID)
			if cleanResolved == cleanReceived {
				match = true
			}
		}
	}

	if !match {
		return fmt.Errorf("nomor handphone Anda tidak cocok dengan profil akun PEKAN")
	}

	// 2. Delete any existing sessions for this phone number or user to allow seamless re-linking/switching
	_ = s.repo.DeleteSessionByPhone(ctx, phoneNumber)
	_ = s.repo.DeleteSessionByUser(ctx, token.TenantID, token.UserID)

	// 3. Create the session
	now := time.Now().UTC()
	err = s.repo.CreateSession(ctx, domain.Session{
		PhoneNumber: phoneNumber,
		TenantID:    token.TenantID,
		UserID:      token.UserID,
		LastActive:  &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// 4. Cleanup the used token
	_ = s.repo.DeleteOTPToken(ctx, code)

	return nil
}

type parsedTransaction struct {
	Action       string            `json:"action"`
	TargetID     string            `json:"target_id"`
	Amount       int64             `json:"amount"`
	Type         string            `json:"type"`
	Description  string            `json:"description"`
	CategoryName string            `json:"category_name"`
	Items        []domain.ChatItem `json:"items"`
	Date         string            `json:"date"`
}

func parseTransactionPrompt(todayDate string) string {
	return fmt.Sprintf(`You are PEKAN AI Assistant, an expert in Indonesian natural language transaction parsing and action execution.
Analyze the user's message carefully to extract transaction details. The user might write one or multiple transactions/actions in a single message.

Return a JSON ARRAY of objects, even if there is only one transaction. If the message is NOT a transaction or action request (like general chat, greeting, or report inquiry), return an empty JSON array: [].

Each object in the JSON array MUST follow this schema:
{
  "action": "create", // "create" to record a new transaction, "delete" to delete/cancel a transaction by ID, "update" to change/edit a transaction by ID
  "target_id": "string", // if the action is "delete" or "update" and the user mentions a transaction ID (e.g. TX-c5e8211b or c5e8211b), put the cleaned hexadecimal part here, otherwise empty
  "amount": 0, // final total amount of the transaction (for create/update)
  "type": "expense", // "expense", "income", "transfer", or "savings"
  "description": "string describing the main transaction",
  "category_name": "string suggested category name (e.g., Makanan, Gaji, Transportasi, Belanja, Cicilan)",
  "items": [{"name":"string","qty":1,"price":0}], // list of items if specified, otherwise empty array
  "date": "YYYY-MM-DD" // transaction date. If the user mentions a specific date or relative date (like "yesterday", "kemarin", "2 hari lalu", "24 april", or "24 april 2026"), calculate the target date using today's date [%s] as the baseline reference. If no date is mentioned at all, use today's date [%s] as the default date.
}

IMPORTANT RULES:
- "action" MUST be one of "create", "delete", "update".
- "type" MUST be one of "expense", "income", "transfer", "savings".
- For Indonesian currency representation: "rb" or "k" means thousand (e.g., 8rb = 8000, 10k = 10000), "jt" means million (e.g., 1.5jt = 1500000).
- Do not include markdown fences. Return PURE JSON only.`, todayDate, todayDate)
}

func formatRupiah(amount int64) string {
	str := fmt.Sprintf("%d", amount)
	var result []string
	length := len(str)
	for i := length; i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		result = append([]string{str[start:i]}, result...)
	}
	return "Rp " + strings.Join(result, ".")
}

func (s *Service) parseTransactionWithLLM(ctx context.Context, message string) ([]parsedTransaction, error) {
	providerCode, err := s.settings.GetGlobalSettingRaw(ctx, "receipt_active_ai_provider")
	if err != nil || providerCode == "" {
		providerCode = "gemini" // fallback
	}
	providerCode = strings.ToLower(strings.TrimSpace(providerCode))

	apiKey, err := s.settings.GetGlobalSettingRaw(ctx, "receipt_api_key_" + providerCode)
	if err != nil || apiKey == "" {
		apiKey, _ = s.settings.GetGlobalSettingRaw(ctx, "receipt_api_key_gemini")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("AI Provider API key not configured")
	}

	// Normalize anthropic/claude aliases
	if providerCode == "claude" {
		providerCode = "anthropic"
	}

	modelName, _ := s.settings.GetGlobalSettingRaw(ctx, "receipt_model_" + providerCode)
	if modelName == "" {
		switch providerCode {
		case "openai":
			modelName = "gpt-4o-mini"
		case "anthropic":
			modelName = "claude-3-5-sonnet-20240620"
		default:
			modelName = "gemini-2.0-flash"
		}
	}

	todayDate := time.Now().Format("2006-01-02")
	var jsonText string
	client := &http.Client{Timeout: 45 * time.Second}

	if providerCode == "gemini" {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
		payload := map[string]any{
			"contents": []any{map[string]any{"parts": []any{
				map[string]any{"text": parseTransactionPrompt(todayDate)},
				map[string]any{"text": fmt.Sprintf("User Message: %q", message)},
			}}},
			"generationConfig": map[string]any{"temperature": 0, "responseMimeType": "application/json"},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("gemini error (status %d): %s", resp.StatusCode, string(raw))
		}
		
		var parsed struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return nil, err
		}
		if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
			return nil, fmt.Errorf("gemini returned no content candidates")
		}
		jsonText = parsed.Candidates[0].Content.Parts[0].Text
	} else if providerCode == "anthropic" {
		url := "https://api.anthropic.com/v1/messages"
		payload := map[string]any{
			"model":      modelName,
			"max_tokens": 2048,
			"system":     parseTransactionPrompt(todayDate),
			"messages": []any{
				map[string]any{"role": "user", "content": fmt.Sprintf("User Message: %q", message)},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("anthropic error (status %d): %s", resp.StatusCode, string(raw))
		}
		
		var parsed struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return nil, err
		}
		if len(parsed.Content) == 0 {
			return nil, fmt.Errorf("anthropic returned no content")
		}
		jsonText = parsed.Content[0].Text
	} else if providerCode == "openai" || providerCode == "sumopod" || providerCode == "openai_compatible" {
		baseURL := "https://api.openai.com/v1"
		if providerCode == "sumopod" {
			baseURL = "https://ai.sumopod.com/v1"
		}
		
		url := baseURL + "/chat/completions"
		payload := map[string]any{
			"model":       modelName,
			"temperature": 0,
			"messages": []any{
				map[string]any{"role": "system", "content": parseTransactionPrompt(todayDate)},
				map[string]any{"role": "user", "content": fmt.Sprintf("User Message: %q", message)},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("openai compatible error (status %d): %s", resp.StatusCode, string(raw))
		}
		
		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return nil, err
		}
		if len(parsed.Choices) == 0 {
			return nil, fmt.Errorf("openai compatible returned no content choices")
		}
		jsonText = parsed.Choices[0].Message.Content
	} else {
		return nil, fmt.Errorf("unsupported provider code: %s", providerCode)
	}

	jsonText = strings.TrimSpace(jsonText)
	fencedJSONRegex := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\}|\\[.*?\\])\\s*```")
	if matches := fencedJSONRegex.FindStringSubmatch(jsonText); len(matches) > 1 {
		jsonText = strings.TrimSpace(matches[1])
	}
	
	firstObject := strings.Index(jsonText, "{")
	firstArray := strings.Index(jsonText, "[")
	
	start := -1
	end := -1
	if firstArray >= 0 && (firstObject < 0 || firstArray < firstObject) {
		start = firstArray
		end = strings.LastIndex(jsonText, "]")
	} else if firstObject >= 0 {
		start = firstObject
		end = strings.LastIndex(jsonText, "}")
	}
	
	if start >= 0 && end > start {
		jsonText = jsonText[start : end+1]
	}

	jsonText = strings.TrimSpace(jsonText)
	// Handle raw multiple blocks like []\n[]
	if strings.Contains(jsonText, "]\n[") || strings.Contains(jsonText, "]\r\n[") {
		idx := strings.Index(jsonText, "]")
		if idx >= 0 {
			jsonText = jsonText[:idx+1]
		}
	}

	var out []parsedTransaction
	if err := json.Unmarshal([]byte(jsonText), &out); err != nil {
		wrappedJSON := "[" + jsonText + "]"
		var singleOut []parsedTransaction
		if err2 := json.Unmarshal([]byte(wrappedJSON), &singleOut); err2 == nil {
			return singleOut, nil
		}
		return nil, fmt.Errorf("failed to parse JSON: %w. Raw: %s", err, jsonText)
	}
	return out, nil
}

func (s *Service) logJSON(level, event string, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["level"] = level
	fields["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	fields["module"] = "whatsapp_usecase"
	fields["event"] = event
	bytes, err := json.Marshal(fields)
	if err == nil {
		fmt.Println(string(bytes))
	}
}

// ProcessAIChat acts as the natural language parser bridge. 
// It parses the transaction using the active AI provider, creates it in the database, and returns a reply.
func (s *Service) ProcessAIChat(ctx context.Context, phoneNumber, message string, tenantID, userID string) (string, error) {
	// Clean JID suffixes (e.g. "@lid", "@c.us") to lookup the session properly
	cleanPhone := phoneNumber
	if strings.Contains(phoneNumber, "@") {
		cleanPhone = strings.Split(phoneNumber, "@")[0]
	}

	var session domain.Session
	var err error
	if tenantID != "" && userID != "" {
		session, err = s.repo.GetSessionByUser(ctx, tenantID, userID)
	} else {
		session, err = s.repo.GetSessionByPhone(ctx, cleanPhone)
	}
	if err != nil {
		return "⚠️ *Nomor Anda Belum Terdaftar*\n\n" +
			"Nomor WhatsApp Anda belum terhubung dengan akun PEKAN.\n\n" +
			"Silakan generate kode OTP terlebih dahulu di menu *Integration* pada dasbor PEKAN, " +
			"lalu kirimkan kode tersebut di sini dengan format:\n" +
			"`!login <KODE_OTP>`", nil
	}

	// Override cleanPhone with the actual session phone number so that user-level pending scans work correctly
	cleanPhone = session.PhoneNumber

	// Ping the session to keep it fresh
	_ = s.repo.UpdateLastActive(ctx, cleanPhone)

	// 1. Fetch tenant code
	tenantCode, err := s.repo.GetTenantCode(ctx, session.TenantID)
	if err != nil {
		s.logJSON("error", "fetch_tenant_code_failed", map[string]any{"phone_number": phoneNumber, "error": err.Error()})
		return "⚠️ *Gagal Menyimpan Transaksi*\n\n" +
			"Maaf, terjadi kesalahan internal saat menghubungkan ke workspace Anda.\n" +
			"Silakan coba lagi beberapa saat lagi.", nil
	}

	cleanMsg := strings.ToLower(strings.TrimSpace(message))

	// FIRST: Check if there is a pending confirmation for this sender
	s.pendingMu.Lock()
	pending, hasPending := s.pendingScans[cleanPhone]
	if hasPending && time.Now().After(pending.ExpiresAt) {
		delete(s.pendingScans, cleanPhone)
		hasPending = false
	}
	s.pendingMu.Unlock()

	if hasPending {
		if cleanMsg == "ya" || cleanMsg == "ok" || cleanMsg == "yes" || cleanMsg == "oke" || cleanMsg == "benar" {
			s.pendingMu.Lock()
			delete(s.pendingScans, cleanPhone)
			s.pendingMu.Unlock()

			txID, err := s.repo.CreateChatTransaction(ctx, session.TenantID, session.UserID, tenantCode, pending.Amount, pending.Type, pending.Description, pending.CategoryName, pending.Date, pending.Items)
			if err != nil {
				return "⚠️ *Gagal Mencatat Transaksi*\n\nMaaf, terjadi kesalahan saat menyimpan transaksi.", nil
			}

			shortID := txID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}

			typeLabel := "Pengeluaran"
			if pending.Type == "income" {
				typeLabel = "Pemasukan"
			} else if pending.Type == "transfer" {
				typeLabel = "Transfer"
			} else if pending.Type == "savings" {
				typeLabel = "Tabungan"
			}

			return fmt.Sprintf("✅ *Transaksi Berhasil Dicatat!*\n\n• *ID*: %s\n• *Tipe*: %s\n• *Total*: %s\n• *Kategori*: %s\n• *Tanggal*: %s\n• *Deskripsi*: %s",
				shortID, typeLabel, formatRupiah(pending.Amount), pending.CategoryName, pending.Date, pending.Description), nil
		} else if cleanMsg == "batal" || cleanMsg == "cancel" || cleanMsg == "tidak" || cleanMsg == "no" {
			s.pendingMu.Lock()
			delete(s.pendingScans, cleanPhone)
			s.pendingMu.Unlock()
			return "❌ *Scanning Dibatalkan.* Transaksi tidak dicatat.", nil
		}
	}

	// SECOND: Check if the command is a !scan command
	if strings.HasPrefix(cleanMsg, "!scan") {
		// Extract URL from message
		var mediaURL string
		parts := strings.Fields(message)
		for _, part := range parts {
			if strings.HasPrefix(strings.ToLower(part), "http://") || strings.HasPrefix(strings.ToLower(part), "https://") {
				mediaURL = part
				break
			}
		}

		if mediaURL == "" {
			return "⚠️ *Gagal Melakukan Scanning*\n\nHarap kirimkan foto/struk belanja Anda dengan menambahkan caption `!scan` saat mengirimkannya, atau gunakan perintah:\n`!scan <URL_GAMBAR>`", nil
		}

		// Download image
		imgBytes, mimeType, err := downloadImage(ctx, mediaURL)
		if err != nil {
			s.logJSON("error", "download_whatsapp_media_failed", map[string]any{"phone_number": phoneNumber, "url": mediaURL, "error": err.Error()})
			return "⚠️ *Gagal Mengunduh Gambar*\n\nMaaf, asisten AI gagal mengunduh gambar struk Anda dari server WhatsApp. Silakan coba lagi beberapa saat lagi.", nil
		}

		// Analyze with LLM
		parsed, err := s.scanReceiptWithLLM(ctx, imgBytes, mimeType)
		if err != nil {
			s.logJSON("error", "whatsapp_receipt_scan_failed", map[string]any{"phone_number": phoneNumber, "error": err.Error()})
			return "⚠️ *Gagal Menganalisis Struk*\n\nMaaf, asisten AI gagal menganalisis struk belanja Anda. Pastikan gambar cukup jelas dan coba lagi.", nil
		}

		amountMinor := int64(parsed.Total)
		if amountMinor <= 0 {
			amountMinor = 0
		}

		pType := strings.ToLower(strings.TrimSpace(parsed.SuggestedType))
		if pType == "" {
			pType = "expense"
		}

		pCat := strings.TrimSpace(parsed.SuggestedCategoryName)
		if pCat == "" {
			pCat = "Belanja"
		}

		pDesc := strings.TrimSpace(parsed.Description)
		if pDesc == "" {
			if parsed.MerchantName != "" {
				pDesc = "Belanja di " + parsed.MerchantName
			} else {
				pDesc = "Transaksi dari Struk"
			}
		}

		pDate := strings.TrimSpace(parsed.TransactionDate)
		if pDate == "" {
			pDate = time.Now().Format("2006-01-02")
		}

		// Save to pending map
		s.pendingMu.Lock()
		s.pendingScans[cleanPhone] = pendingTransaction{
			Amount:       amountMinor,
			Type:         pType,
			Description:  pDesc,
			CategoryName: pCat,
			Date:         pDate,
			ExpiresAt:    time.Now().Add(5 * time.Minute),
		}
		s.pendingMu.Unlock()

		typeLabel := "Pengeluaran"
		if pType == "income" {
			typeLabel = "Pemasukan"
		} else if pType == "transfer" {
			typeLabel = "Transfer"
		} else if pType == "savings" {
			typeLabel = "Tabungan"
		}

		return fmt.Sprintf("📄 *Hasil Scan Struk Belanja:*\n\n"+
			"• *Tipe*: %s\n"+
			"• *Total*: %s\n"+
			"• *Kategori*: %s\n"+
			"• *Tanggal*: %s\n"+
			"• *Deskripsi*: %s\n\n"+
			"Apakah detail transaksi di atas sudah benar?\n\n"+
			"Balas dengan *YA* atau *OK* untuk langsung mencatat transaksi ini ke keuangan Anda.",
			typeLabel, formatRupiah(amountMinor), pCat, pDate, pDesc), nil
	}

	// 2. Parse the chat message using LLM
	parsedList, err := s.parseTransactionWithLLM(ctx, message)
	if err != nil {
		s.logJSON("error", "parse_transaction_failed", map[string]any{"phone_number": phoneNumber, "error": err.Error()})
		parsedList = nil
	}

	var replies []string
	var successCount int

	for _, parsed := range parsedList {
		if parsed.Action == "delete" && parsed.TargetID != "" {
			txItem, err := s.repo.FindChatTransaction(ctx, session.TenantID, session.UserID, tenantCode, parsed.TargetID)
			if err != nil {
				replies = append(replies, fmt.Sprintf("⚠️ Transaksi ID *%s* tidak ditemukan.", parsed.TargetID))
				continue
			}
			err = s.repo.DeleteChatTransaction(ctx, session.TenantID, session.UserID, tenantCode, parsed.TargetID)
			if err != nil {
				replies = append(replies, fmt.Sprintf("⚠️ Gagal menghapus transaksi ID *%s*.", parsed.TargetID))
				continue
			}
			replies = append(replies, fmt.Sprintf("🗑️ *Dihapus (ID: %s)*: %s (%s)", txItem.ID[:8], txItem.Description, formatRupiah(txItem.Amount)))
			successCount++
			continue
		}

		if parsed.Action == "update" && parsed.TargetID != "" {
			txItem, err := s.repo.FindChatTransaction(ctx, session.TenantID, session.UserID, tenantCode, parsed.TargetID)
			if err != nil {
				replies = append(replies, fmt.Sprintf("⚠️ Transaksi ID *%s* tidak ditemukan.", parsed.TargetID))
				continue
			}
			newAmount := parsed.Amount
			if newAmount <= 0 {
				newAmount = txItem.Amount
			}
			newType := parsed.Type
			if newType == "" {
				newType = txItem.Type
			}
			newDesc := parsed.Description
			if newDesc == "" {
				newDesc = txItem.Description
			}
			newCat := parsed.CategoryName
			if newCat == "" {
				newCat = txItem.CategoryName
			}
			newDate := parsed.Date
			if newDate == "" {
				newDate = txItem.TxDate
			}

			err = s.repo.UpdateChatTransaction(ctx, session.TenantID, session.UserID, tenantCode, parsed.TargetID, newAmount, newType, newDesc, newCat, newDate)
			if err != nil {
				replies = append(replies, fmt.Sprintf("⚠️ Gagal memperbarui transaksi ID *%s*.", parsed.TargetID))
				continue
			}
			replies = append(replies, fmt.Sprintf("✏️ *Diperbarui (ID: %s)*: %s (%s, Tanggal: %s)", txItem.ID[:8], newDesc, formatRupiah(newAmount), newDate))
			successCount++
			continue
		}

		// Otherwise, it's a create action
		if parsed.Amount <= 0 {
			continue // skip empty/invalid transactions
		}

		txID, err := s.repo.CreateChatTransaction(ctx, session.TenantID, session.UserID, tenantCode, parsed.Amount, parsed.Type, parsed.Description, parsed.CategoryName, parsed.Date, parsed.Items)
		if err != nil {
			s.logJSON("error", "create_transaction_failed", map[string]any{"phone_number": phoneNumber, "error": err.Error()})
			replies = append(replies, fmt.Sprintf("⚠️ Gagal mencatat *%s*.", parsed.Description))
			continue
		}

		typeLabel := "Pengeluaran"
		if parsed.Type == "income" {
			typeLabel = "Pemasukan"
		} else if parsed.Type == "transfer" {
			typeLabel = "Transfer"
		} else if parsed.Type == "savings" {
			typeLabel = "Tabungan"
		}

		shortID := txID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		replies = append(replies, fmt.Sprintf("✅ *Dicatat (ID: %s)*: %s *%s* (%s, Tanggal: %s)", shortID, typeLabel, formatRupiah(parsed.Amount), parsed.Description, parsed.Date))
		successCount++
	}

	if successCount == 0 && len(replies) == 0 {
		s.logJSON("info", "non_transaction_message_received", map[string]any{"phone_number": phoneNumber, "message": message, "parsed_amount": 0})
		
		finContext, err := s.repo.GetFinancialContext(ctx, session.TenantID, session.UserID, tenantCode)
		if err == nil && finContext != nil {
			aiResponse, err := s.generateInteractiveResponseWithLLM(ctx, message, finContext)
			if err == nil && aiResponse != "" {
				return aiResponse, nil
			}
			s.logJSON("warn", "generate_interactive_response_failed", map[string]any{"phone_number": phoneNumber, "error": err.Error()})
		} else {
			s.logJSON("warn", "fetch_financial_context_failed", map[string]any{"phone_number": phoneNumber, "error": err.Error()})
		}

		reply := "👋 *Halo! Saya Asisten AI PEKAN.*\n\n" +
			"Saya siap membantu Anda mencatat transaksi keuangan secara otomatis langsung dari WhatsApp.\n\n" +
			"💡 *Bagaimana cara mencatat?*\n" +
			"Cukup kirimkan pesan seperti:\n" +
			"• _\"catat pengeluaran bensin 50rb\"_\n" +
			"• _\"hapus transaksi c5e8211b\"_\n" +
			"• _\"ubah transaksi c5e8211b nominalnya jadi 75rb\"_\n\n" +
			"Anda juga dapat menanyakan rangkuman keuangan Anda di sini!"
		return reply, nil
	}

	var sb strings.Builder
	sb.WriteString("*PEKAN Asisten AI*\n\n")
	sb.WriteString("Berhasil memproses permintaan Anda:\n")
	for _, rep := range replies {
		sb.WriteString(fmt.Sprintf("- %s\n", rep))
	}
	sb.WriteString("\n_Pesan Anda diproses oleh Asisten AI PEKAN_")
	return sb.String(), nil
}

func (s *Service) generateInteractiveResponseWithLLM(ctx context.Context, message string, finContext *domain.FinancialContext) (string, error) {
	providerCode, err := s.settings.GetGlobalSettingRaw(ctx, "wa_bot_active_ai_provider")
	if err != nil || providerCode == "" {
		providerCode, _ = s.settings.GetGlobalSettingRaw(ctx, "receipt_active_ai_provider")
		if providerCode == "" {
			providerCode = "gemini"
		}
	}
	providerCode = strings.ToLower(strings.TrimSpace(providerCode))
	// Normalize anthropic/claude aliases
	if providerCode == "claude" {
		providerCode = "anthropic"
	}

	apiKey, err := s.settings.GetGlobalSettingRaw(ctx, "receipt_api_key_" + providerCode)
	if err != nil || apiKey == "" {
		apiKey, _ = s.settings.GetGlobalSettingRaw(ctx, "receipt_api_key_gemini")
	}

	if apiKey == "" {
		return "", fmt.Errorf("AI Provider API key not configured")
	}

	modelName, _ := s.settings.GetGlobalSettingRaw(ctx, "wa_bot_model_" + providerCode)
	if modelName == "" {
		modelName, _ = s.settings.GetGlobalSettingRaw(ctx, "receipt_model_" + providerCode)
		if modelName == "" {
			switch providerCode {
			case "openai":
				modelName = "gpt-4o-mini"
			case "anthropic":
				modelName = "claude-3-5-sonnet-20240620"
			default:
				modelName = "gemini-2.0-flash"
			}
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	instructions, _ := s.settings.GetGlobalSettingRaw(ctx, "wa_bot_system_instructions")
	systemPrompt := waBotSystemPrompt(finContext, instructions)

	if providerCode == "gemini" {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
		payload := map[string]any{
			"systemInstruction": map[string]any{
				"parts": []any{
					map[string]any{"text": systemPrompt},
				},
			},
			"contents": []any{
				map[string]any{"role": "user", "parts": []any{
					map[string]any{"text": message},
				}},
			},
			"generationConfig": map[string]any{"temperature": 0.7},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("gemini error (status %d): %s", resp.StatusCode, string(raw))
		}
		
		var parsed struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return "", err
		}
		if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("gemini returned no content candidates")
		}
		return parsed.Candidates[0].Content.Parts[0].Text, nil
	} else if providerCode == "openai" || providerCode == "sumopod" || providerCode == "openai_compatible" {
		baseURL := "https://api.openai.com/v1"
		if providerCode == "sumopod" {
			baseURL = "https://ai.sumopod.com/v1"
		}
		
		url := baseURL + "/chat/completions"
		payload := map[string]any{
			"model":       modelName,
			"temperature": 0.7,
			"messages": []any{
				map[string]any{"role": "system", "content": systemPrompt},
				map[string]any{"role": "user", "content": message},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("openai-compatible error (status %d): %s", resp.StatusCode, string(raw))
		}
		
		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return "", err
		}
		if len(parsed.Choices) == 0 {
			return "", fmt.Errorf("openai-compatible returned empty choices")
		}
		return parsed.Choices[0].Message.Content, nil
	} else if providerCode == "anthropic" {
		url := "https://api.anthropic.com/v1/messages"
		payload := map[string]any{
			"model":      modelName,
			"max_tokens": 1024,
			"system":     systemPrompt,
			"messages": []any{
				map[string]any{"role": "user", "content": message},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("claude error (status %d): %s", resp.StatusCode, string(raw))
		}
		
		var parsed struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return "", err
		}
		if len(parsed.Content) == 0 {
			return "", fmt.Errorf("claude returned empty content")
		}
		return parsed.Content[0].Text, nil
	}

	return "", fmt.Errorf("unsupported AI provider: %s", providerCode)
}

func waBotSystemPrompt(finContext *domain.FinancialContext, instructions string) string {
	var sb strings.Builder
	if instructions != "" {
		sb.WriteString(instructions)
	} else {
		sb.WriteString("Anda adalah Asisten AI PEKAN, perencana keuangan pribadi yang profesional, ringkas, dan sangat membantu.\n")
		sb.WriteString("Tugas Anda adalah membalas pesan pengguna WhatsApp secara interaktif. Pengguna sudah login/terverifikasi.\n\n")
		sb.WriteString("--- ATURAN BERKOMUNIKASI ---\n")
		sb.WriteString("1. Jawablah menggunakan bahasa Indonesia yang natural, profesional, sopan, dan langsung pada intinya (to the point).\n")
		sb.WriteString("2. HINDARI mengulang-ulang sapaan formal pembuka yang sama (seperti \"Halo! Selamat siang/sore/malam. Senang sekali bisa membantu...\" atau \"Sebagai Asisten AI PEKAN...\") di setiap pesan. Langsung jawab pertanyaan pengguna secara spesifik.\n")
		sb.WriteString("3. Jika pengguna menyapa singkat (seperti 'halo' atau 'hai'), sapa balik secara singkat, bersahabat, dan ingatkan secara ringkas bahwa Anda dapat membantu mencatat transaksi (misal: 'catat pengeluaran bensin 20rb') atau membacakan laporan keuangan.\n")
		sb.WriteString("4. Jika pengguna menanyakan sisa anggaran, pengeluaran, pemasukan, atau laporan transaksi, bacakan data rill di bawah ini secara akurat. Tampilkan data dengan rapi menggunakan poin-poin terstruktur agar mudah dibaca.\n")
		sb.WriteString("5. Berikan saran atau rekomendasi finansial secara cerdas, realistis, dan memotivasi tanpa menggurui.\n")
		sb.WriteString("6. Gunakan format tebal (bold) WhatsApp dengan tanda bintang (*) untuk hal-hal penting seperti kategori, nominal rupiah, atau sisa anggaran agar nyaman dibaca di layar HP.\n")
		sb.WriteString("7. Jangan menyebutkan bahwa Anda adalah model bahasa besar. Berperanlah 100% sebagai Asisten AI PEKAN.\n")
	}

	sb.WriteString("\n\nBerikut adalah DATA KEUANGAN RILL pengguna saat ini (Gunakan data ini untuk menjawab pertanyaan finansial mereka):\n")
	sb.WriteString("--- SUMMARY BULAN INI ---\n")
	sb.WriteString(fmt.Sprintf("- Total Pemasukan: Rp %s\n", formatRupiah(finContext.TotalIncome)))
	sb.WriteString(fmt.Sprintf("- Total Pengeluaran: Rp %s\n", formatRupiah(finContext.TotalExpense)))
	sb.WriteString(fmt.Sprintf("- Selisih (Arus Kas): Rp %s\n\n", formatRupiah(finContext.TotalIncome - finContext.TotalExpense)))

	sb.WriteString("--- DAFTAR ANGGARAN (BUDGET) ---\n")
	if len(finContext.ActiveBudgets) == 0 {
		sb.WriteString("- Belum ada anggaran aktif yang dibuat.\n\n")
	} else {
		for _, b := range finContext.ActiveBudgets {
			sb.WriteString(fmt.Sprintf("- Anggaran %s (%s): Batas Rp %s, Terpakai Rp %s (Sisa Rp %s)\n",
				b.Name, b.CategoryName, formatRupiah(b.AmountLimitMinor), formatRupiah(b.SpentAmountMinor),
				formatRupiah(b.AmountLimitMinor - b.SpentAmountMinor)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("--- TRANSAKSI TERAKHIR (MAKSIMAL 100) ---\n")
	if len(finContext.RecentTx) == 0 {
		sb.WriteString("- Belum ada riwayat transaksi.\n\n")
	} else {
		for _, t := range finContext.RecentTx {
			typeLabel := "Pengeluaran"
			if t.Type == "income" {
				typeLabel = "Pemasukan"
			}
			shortID := t.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			sb.WriteString(fmt.Sprintf("- [%s] %s: Rp %s (%s, Kategori: %s) - *ID: %s*\n",
				t.TxDate, typeLabel, formatRupiah(t.Amount), t.Description, t.CategoryName, shortID))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (s *Service) EnqueueMessage(ctx context.Context, phoneNumber, message string, tenantID, userID *string) (string, error) {
	return s.repo.EnqueueMessage(ctx, phoneNumber, message, tenantID, userID)
}

func (s *Service) StartQueueWorker(ctx context.Context, numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					s.processNextQueueItem(ctx)
				}
			}
		}(i)
	}
}

func (s *Service) processNextQueueItem(ctx context.Context) {
	// Retrieve 1 pending queue item
	items, err := s.repo.GetPendingQueueItems(ctx, 1)
	if err != nil || len(items) == 0 {
		return
	}
	item := items[0]

	// Mark it as processing
	err = s.repo.UpdateQueueItemStatus(ctx, item.ID, "processing", nil, nil, nil)
	if err != nil {
		return
	}

	s.logJSON("info", "queue_item_processing", map[string]any{
		"id":           item.ID,
		"phone_number": item.PhoneNumber,
		"message":      item.Message,
	})

	startTime := time.Now()
	var replyMsg string
	var errMsg string

	// Run ProcessAIChat with a timeout
	chatCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var tID, uID string
	if item.TenantID != nil {
		tID = *item.TenantID
	}
	if item.UserID != nil {
		uID = *item.UserID
	}
	replyMsg, err = s.ProcessAIChat(chatCtx, item.PhoneNumber, item.Message, tID, uID)
	latency := int(time.Since(startTime).Milliseconds())

	if err != nil {
		errMsg = err.Error()
		statusErr := s.repo.UpdateQueueItemStatus(ctx, item.ID, "failed", nil, &errMsg, &latency)
		if statusErr != nil {
			s.logJSON("error", "queue_status_update_failed", map[string]any{"id": item.ID, "error": statusErr.Error()})
		}

		s.logJSON("error", "queue_item_processed_failed", map[string]any{
			"id":           item.ID,
			"phone_number": item.PhoneNumber,
			"latency_ms":   latency,
			"error":        errMsg,
		})
		
		// Send a friendly error message to the user on failure
		fallback := "Maaf, terjadi kesalahan saat memproses pesan Anda. Asisten AI kami sedang mengalami kendala jaringan. Silakan coba kirimkan kembali pesan Anda dalam beberapa saat lagi."
		_ = s.SendWhatsAppMessage(ctx, item.PhoneNumber, fallback)
		return
	}

	// Send reply
	if replyMsg != "" {
		err = s.SendWhatsAppMessage(ctx, item.PhoneNumber, replyMsg)
		if err != nil {
			errMsg = fmt.Sprintf("failed to send: %v", err)
			_ = s.repo.UpdateQueueItemStatus(ctx, item.ID, "failed", &replyMsg, &errMsg, &latency)
			
			s.logJSON("error", "queue_item_send_failed", map[string]any{
				"id":           item.ID,
				"phone_number": item.PhoneNumber,
				"latency_ms":   latency,
				"error":        errMsg,
			})
			return
		}
	}

	// Success
	_ = s.repo.UpdateQueueItemStatus(ctx, item.ID, "success", &replyMsg, nil, &latency)
	s.logJSON("info", "queue_item_processed_success", map[string]any{
		"id":           item.ID,
		"phone_number": item.PhoneNumber,
		"latency_ms":   latency,
		"reply":        replyMsg,
	})
}

func (s *Service) GetSessionByPhone(ctx context.Context, phoneNumber string) (domain.Session, error) {
	return s.repo.GetSessionByPhone(ctx, phoneNumber)
}

type parsedReceipt struct {
	MerchantName          string  `json:"merchant_name"`
	Total                 float64 `json:"total"`
	SuggestedType         string  `json:"suggested_type"`
	SuggestedCategoryName string  `json:"suggested_category_name"`
	TransactionDate       string  `json:"transaction_date"`
	Description           string  `json:"description"`
}

func downloadImage(ctx context.Context, urlStr string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download image, status: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg" // fallback
	}
	return data, mimeType, nil
}

func (s *Service) scanReceiptWithLLM(ctx context.Context, content []byte, mimeType string) (parsedReceipt, error) {
	providerCode, err := s.settings.GetGlobalSettingRaw(ctx, "receipt_active_ai_provider")
	if err != nil || providerCode == "" {
		providerCode = "gemini" // fallback
	}
	providerCode = strings.ToLower(strings.TrimSpace(providerCode))
	// Normalize anthropic/claude aliases
	if providerCode == "claude" {
		providerCode = "anthropic"
	}

	apiKey, err := s.settings.GetGlobalSettingRaw(ctx, "receipt_api_key_" + providerCode)
	if err != nil || apiKey == "" {
		apiKey, _ = s.settings.GetGlobalSettingRaw(ctx, "receipt_api_key_gemini")
	}

	if apiKey == "" {
		return parsedReceipt{}, fmt.Errorf("AI Provider API key not configured")
	}

	modelName, _ := s.settings.GetGlobalSettingRaw(ctx, "receipt_model_" + providerCode)
	if modelName == "" {
		switch providerCode {
		case "openai":
			modelName = "gpt-4o-mini"
		case "anthropic":
			modelName = "claude-3-5-sonnet-20240620"
		default:
			modelName = "gemini-2.0-flash"
		}
	}

	var jsonText string
	client := &http.Client{Timeout: 30 * time.Second}

	prompt := `You are PEKAN AI Assistant, an expert in Indonesian receipt analysis.
Analyze the receipt image carefully and extract these fields:
1. merchant_name (string, name of the store/merchant)
2. total (number, final total amount of transaction)
3. suggested_type (string, "expense" or "income" or "transfer" or "savings")
4. suggested_category_name (string, e.g. Makanan, Belanja, Transportasi)
5. transaction_date (string, YYYY-MM-DD format)
6. description (string, describing the transaction)

Return JSON only with this schema:
{
  "merchant_name": "string",
  "total": 0,
  "suggested_type": "expense|income|transfer|savings",
  "suggested_category_name": "string",
  "transaction_date": "YYYY-MM-DD",
  "description": "string"
}
Do not include markdown fences. Return PURE JSON only.`

	if providerCode == "gemini" {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
		payload := map[string]any{
			"contents": []any{map[string]any{"parts": []any{
				map[string]any{"text": prompt},
				map[string]any{"inline_data": map[string]any{
					"mime_type": mimeType,
					"data":      base64.StdEncoding.EncodeToString(content),
				}},
			}}},
			"generationConfig": map[string]any{"temperature": 0, "responseMimeType": "application/json"},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			return parsedReceipt{}, err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			return parsedReceipt{}, fmt.Errorf("gemini error (status %d): %s", resp.StatusCode, string(raw))
		}
		
		var parsed struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return parsedReceipt{}, err
		}
		if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
			return parsedReceipt{}, fmt.Errorf("gemini returned no content candidates")
		}
		jsonText = parsed.Candidates[0].Content.Parts[0].Text
	} else if providerCode == "anthropic" {
		url := "https://api.anthropic.com/v1/messages"
		payload := map[string]any{
			"model":      modelName,
			"max_tokens": 2048,
			"system":     prompt,
			"messages": []any{
				map[string]any{"role": "user", "content": []any{
					map[string]any{"type": "text", "text": "Read this shopping receipt image and extract the fields."},
					map[string]any{"type": "image", "source": map[string]any{
						"type":       "base64",
						"media_type": mimeType,
						"data":       base64.StdEncoding.EncodeToString(content),
					}},
				}},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			return parsedReceipt{}, err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			return parsedReceipt{}, fmt.Errorf("anthropic error (status %d): %s", resp.StatusCode, string(raw))
		}
		
		var parsed struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return parsedReceipt{}, err
		}
		if len(parsed.Content) == 0 {
			return parsedReceipt{}, fmt.Errorf("anthropic returned no content")
		}
		jsonText = parsed.Content[0].Text
	} else if providerCode == "openai" || providerCode == "sumopod" || providerCode == "openai_compatible" {
		baseURL := "https://api.openai.com/v1"
		if providerCode == "sumopod" {
			baseURL = "https://ai.sumopod.com/v1"
		}
		
		url := baseURL + "/chat/completions"
		payload := map[string]any{
			"model":       modelName,
			"temperature": 0,
			"messages": []any{
				map[string]any{"role": "system", "content": prompt},
				map[string]any{"role": "user", "content": []any{
					map[string]any{"type": "text", "text": "Read this shopping receipt image and extract the fields."},
					map[string]any{"type": "image_url", "image_url": map[string]any{"url": fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(content))}},
				}},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			return parsedReceipt{}, err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			return parsedReceipt{}, fmt.Errorf("openai compatible error (status %d): %s", resp.StatusCode, string(raw))
		}
		
		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return parsedReceipt{}, err
		}
		if len(parsed.Choices) == 0 {
			return parsedReceipt{}, fmt.Errorf("openai compatible returned no content choices")
		}
		jsonText = parsed.Choices[0].Message.Content
	} else {
		return parsedReceipt{}, fmt.Errorf("unsupported provider code: %s", providerCode)
	}

	jsonText = strings.TrimSpace(jsonText)
	fencedJSONRegex := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")
	if matches := fencedJSONRegex.FindStringSubmatch(jsonText); len(matches) > 1 {
		jsonText = strings.TrimSpace(matches[1])
	}
	
	firstObject := strings.Index(jsonText, "{")
	end := strings.LastIndex(jsonText, "}")
	if firstObject >= 0 && end > firstObject {
		jsonText = jsonText[firstObject : end+1]
	}

	var out parsedReceipt
	if err := json.Unmarshal([]byte(jsonText), &out); err != nil {
		return parsedReceipt{}, fmt.Errorf("failed to parse JSON: %w. Raw: %s", err, jsonText)
	}
	return out, nil
}

func (s *Service) ProcessWebChat(ctx context.Context, tenantID, userID, message string) (string, error) {
	tenantCode, err := s.repo.GetTenantCode(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get tenant code: %w", err)
	}

	finContext, err := s.repo.GetFinancialContext(ctx, tenantID, userID, tenantCode)
	if err != nil {
		s.logJSON("warn", "web_chat_fetch_financial_context_failed", map[string]any{"tenant_id": tenantID, "user_id": userID, "error": err.Error()})
		finContext = &domain.FinancialContext{}
	}

	reply, err := s.generateInteractiveResponseWithLLM(ctx, message, finContext)
	if err != nil {
		return "", err
	}
	return reply, nil
}

func (s *Service) GetWhatsAppBotNumber(ctx context.Context) string {
	num, _ := s.settings.GetGlobalSettingRaw(ctx, "wa_bot_phone_number")
	return strings.TrimSpace(num)
}

func (s *Service) GetWebhookSecret(ctx context.Context) string {
	secret, _ := s.settings.GetGlobalSettingRaw(ctx, "whatsapp_webhook_secret")
	return strings.TrimSpace(secret)
}



