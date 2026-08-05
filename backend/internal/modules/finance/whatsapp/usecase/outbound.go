package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *Service) SendWhatsAppMessage(ctx context.Context, phone, message string) error {
	if s.settings == nil {
		return fmt.Errorf("konfigurasi notifikasi belum diatur")
	}
	provider, _ := s.settings.GetGlobalSettingRaw(ctx, "notification_wa_active_provider")
	if provider == "" {
		provider = "wa_waha" // Default fallback
	}

	var err error
	switch provider {
	case "wa_fonnte":
		err = s.sendWAViaFonnte(ctx, phone, message)
	case "wa": // Meta/Official
		err = s.sendWAViaMeta(ctx, phone, message)
	default:
		err = s.sendWAViaWAHA(ctx, phone, message)
	}

	if err != nil {
		s.logJSON("error", "send_whatsapp_message_failed", map[string]any{
			"phone_number": phone,
			"provider":     provider,
			"error":        err.Error(),
		})
		return err
	}

	s.logJSON("info", "send_whatsapp_message_success", map[string]any{
		"phone_number": phone,
		"provider":     provider,
		"message":      message,
	})
	return nil
}

func (s *Service) sendWAViaWAHA(ctx context.Context, phone, message string) error {
	configJSON, err := s.settings.GetGlobalSettingRaw(ctx, "notification_wa_waha")
	if err != nil || configJSON == "" {
		return fmt.Errorf("konfigurasi WAHA belum diatur")
	}
	var cfg struct {
		ApiUrl  string `json:"apiUrl"`
		ApiKey  string `json:"apiKey"`
		Session string `json:"session"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil || cfg.ApiUrl == "" {
		return fmt.Errorf("konfigurasi WAHA tidak valid")
	}

	session := cfg.Session
	if session == "" {
		session = "default"
	}

	// Sanitize phone and domain suffix: preserve `@lid` or `@g.us`
	cleanPhone := ""
	domainSuffix := "@c.us"

	if strings.Contains(phone, "@") {
		parts := strings.SplitN(phone, "@", 2)
		phone = parts[0]
		domainSuffix = "@" + parts[1]
	}

	for _, char := range phone {
		if char >= '0' && char <= '9' {
			cleanPhone += string(char)
		}
	}

	chatId := cleanPhone + domainSuffix

	apiUrl := strings.TrimSuffix(cfg.ApiUrl, "/")
	if !strings.Contains(apiUrl, "/api/") {
		apiUrl = apiUrl + "/api/sendText"
	}

	payload, _ := json.Marshal(map[string]any{
		"chatId":  chatId,
		"text":    message,
		"session": session,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", apiUrl, strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	if cfg.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)
		req.Header.Set("X-Api-Key", cfg.ApiKey)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("WAHA API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (s *Service) sendWAViaMeta(ctx context.Context, phone, message string) error {
	configJSON, err := s.settings.GetGlobalSettingRaw(ctx, "notification_wa")
	if err != nil || configJSON == "" {
		return fmt.Errorf("konfigurasi Meta WA belum diatur")
	}
	var cfg struct {
		ApiToken string `json:"apiToken"`
		PhoneID  string `json:"phoneId"`
	}
	json.Unmarshal([]byte(configJSON), &cfg)
	if cfg.ApiToken == "" || cfg.PhoneID == "" {
		return fmt.Errorf("konfigurasi Meta WA tidak valid")
	}

	cleanPhone := sanitizePhoneForMeta(phone)
	apiUrl := fmt.Sprintf("https://graph.facebook.com/v17.0/%s/messages", cfg.PhoneID)

	payload, _ := json.Marshal(map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                cleanPhone,
		"type":              "text",
		"text":              map[string]string{"body": message},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", apiUrl, strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiToken)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Meta API error (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *Service) sendWAViaFonnte(ctx context.Context, phone, message string) error {
	configJSON, err := s.settings.GetGlobalSettingRaw(ctx, "notification_wa_fonnte")
	if err != nil || configJSON == "" {
		return fmt.Errorf("konfigurasi Fonnte belum diatur")
	}
	var cfg struct {
		ApiKey string `json:"apiKey" `
	}
	json.Unmarshal([]byte(configJSON), &cfg)
	if cfg.ApiKey == "" {
		return fmt.Errorf("konfigurasi Fonnte tidak valid")
	}

	cleanPhone := sanitizePhoneForMeta(phone)

	payload, _ := json.Marshal(map[string]any{
		"target":  cleanPhone,
		"message": message,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.fonnte.com/send", strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cfg.ApiKey)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Fonnte API error (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func sanitizePhoneForMeta(phone string) string {
	clean := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			clean += string(char)
		}
	}
	return clean
}
