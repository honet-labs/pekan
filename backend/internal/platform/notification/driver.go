package notification

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

type Driver interface {
	Send(ctx context.Context, destination string, message string) error
}

// SMTP Driver
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Security string `json:"security"` // "none", "ssl", "tls"
}

type SMTPDriver struct {
	Config SMTPConfig
}

func (d *SMTPDriver) Send(ctx context.Context, destination string, message string) error {
	recipients := splitDestinations(destination)
	if len(recipients) == 0 {
		return nil
	}

	addr := fmt.Sprintf("%s:%s", d.Config.Host, d.Config.Port)
	
	dialer := net.Dialer{Timeout: 15 * time.Second}
	var conn net.Conn
	var err error

	// Support Port 465 (Implicit SSL/TLS)
	if d.Config.Security == "ssl" || d.Config.Security == "tls" || d.Config.Port == "465" {
		conn, err = tls.DialWithDialer(&dialer, "tcp", addr, &tls.Config{
			InsecureSkipVerify: true, // Common for many SMTP relays
			ServerName:         d.Config.Host,
		})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}

	if err != nil {
		return fmt.Errorf("dial error: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, d.Config.Host)
	if err != nil {
		return fmt.Errorf("smtp client error: %w", err)
	}
	defer c.Quit()

	auth := smtp.PlainAuth("", d.Config.Username, d.Config.Password, d.Config.Host)
	if err = c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth error: %w", err)
	}

	// Send to all recipients
	for _, to := range recipients {
		if err = c.Mail(d.Config.Username); err != nil {
			return err
		}
		if err = c.Rcpt(to); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		msg := []byte(fmt.Sprintf("To: %s\r\nSubject: Pekan SaaS Notification\r\n\r\n%s", to, message))
		if _, err = w.Write(msg); err != nil {
			return err
		}
		if err = w.Close(); err != nil {
			return err
		}
	}
	
	return nil
}

// Telegram Driver
type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type TelegramDriver struct {
	Config TelegramConfig
}

func (d *TelegramDriver) Send(ctx context.Context, destination string, message string) error {
	targets := splitDestinations(destination)
	if len(targets) == 0 {
		targets = splitDestinations(d.Config.ChatID)
	}
	if len(targets) == 0 {
		return fmt.Errorf("chat_id is required")
	}

	for _, target := range targets {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", d.Config.BotToken)
		payload := map[string]string{
			"chat_id": target,
			"text":    message,
		}
		body, _ := json.Marshal(payload)
		
		req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("telegram api error for %s: status %d", target, resp.StatusCode)
		}
	}
	return nil
}

// WAHA Driver (WhatsApp HTTP API)
type WahaConfig struct {
	ApiUrl  string `json:"apiUrl"`
	ApiKey  string `json:"apiKey"`
	Session string `json:"session"`
}

type WahaDriver struct {
	Config WahaConfig
}

func (d *WahaDriver) Send(ctx context.Context, destination string, message string) error {
	session := d.Config.Session
	if session == "" {
		session = "default"
	}

	targets := splitDestinations(destination)
	for _, target := range targets {
		url := strings.TrimSuffix(d.Config.ApiUrl, "/")
		if !strings.Contains(url, "/api/") {
			url = url + "/api/sendText"
		}

		payload := map[string]any{
			"chatId":  target + "@c.us",
			"text":    message,
			"session": session,
		}
		body, _ := json.Marshal(payload)
		
		req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		if d.Config.ApiKey != "" {
			req.Header.Set("Authorization", "Bearer " + d.Config.ApiKey)
			req.Header.Set("X-Api-Key", d.Config.ApiKey)
		}
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
	}
	return nil
}

// Fonnte Driver
type FonnteConfig struct {
	ApiKey string `json:"apiKey"`
}

type FonnteDriver struct {
	Config FonnteConfig
}

func (d *FonnteDriver) Send(ctx context.Context, destination string, message string) error {
	url := "https://api.fonnte.com/send"
	
	payload := map[string]string{
		"target": destination, // Fonnte supports comma separated in 'target' field natively
		"message": message,
	}
	body, _ := json.Marshal(payload)
	
	req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", d.Config.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 300 {
		return fmt.Errorf("fonnte api error: status %d", resp.StatusCode)
	}
	return nil
}

// GOWA Driver
type GowaConfig struct {
	ApiKey string `json:"apiKey"`
}

type GowaDriver struct {
	Config GowaConfig
}

func (d *GowaDriver) Send(ctx context.Context, destination string, message string) error {
	targets := splitDestinations(destination)
	for _, target := range targets {
		url := "https://api.gowa.id/send-message"
		payload := map[string]string{
			"number": target,
			"message": message,
		}
		body, _ := json.Marshal(payload)
		
		req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
		req.Header.Set("Authorization", "Bearer " + d.Config.ApiKey)
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
	}
	return nil
}

// Meta WhatsApp Driver
type MetaWAConfig struct {
	ApiToken string `json:"apiToken"`
	PhoneID  string `json:"phoneId"`
}

type MetaWADriver struct {
	Config MetaWAConfig
}

func (d *MetaWADriver) Send(ctx context.Context, destination string, message string) error {
	targets := splitDestinations(destination)
	for _, target := range targets {
		url := fmt.Sprintf("https://graph.facebook.com/v17.0/%s/messages", d.Config.PhoneID)
		
		payload := map[string]any{
			"messaging_product": "whatsapp",
			"to":                target,
			"type":              "text",
			"text":              map[string]string{"body": message},
		}
		body, _ := json.Marshal(payload)
		
		req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
		req.Header.Set("Authorization", "Bearer " + d.Config.ApiToken)
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
	}
	return nil
}

func splitDestinations(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	// Support comma or semicolon
	raw = strings.ReplaceAll(raw, ";", ",")
	parts := strings.Split(raw, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}
