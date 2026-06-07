package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"pekan/backend/internal/modules/finance/whatsapp/usecase"
)

type WebhookHandler struct {
	service *usecase.Service
}

func NewWebhookHandler(service *usecase.Service) *WebhookHandler {
	return &WebhookHandler{service: service}
}

func logJSON(level, event string, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["level"] = level
	fields["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	fields["module"] = "whatsapp"
	fields["event"] = event
	
	bytes, err := json.Marshal(fields)
	if err == nil {
		fmt.Println(string(bytes))
	}
}

func (h *WebhookHandler) HandleIncomingMessage(w http.ResponseWriter, r *http.Request) {
	var sender, message string
	var fromJid string
	var mediaURL string
	var participant string
	var raw map[string]interface{}

	// Handle Form Data
	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") || 
	   strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseForm(); err == nil {
			sender = r.FormValue("sender")
			if sender == "" { sender = r.FormValue("phone") }
			message = r.FormValue("message")
			if message == "" { message = r.FormValue("text") }
			mediaURL = r.FormValue("url")
		}
	} else {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			fmt.Printf("[WA-WEBHOOK] RAW BODY: %s\n", string(bodyBytes))
			if err := json.Unmarshal(bodyBytes, &raw); err == nil {
				// Try generic / Fonnte format
				if s, ok := raw["sender"].(string); ok {
					sender = s
				} else if s, ok := raw["phone"].(string); ok {
					sender = s
				}
				if m, ok := raw["message"].(string); ok {
					message = m
				} else if m, ok := raw["text"].(string); ok {
					message = m
				}
				if u, ok := raw["url"].(string); ok {
					mediaURL = u
				}
				if g, ok := raw["group"].(string); ok && g != "" {
					fromJid = g
					if m, ok := raw["member"].(string); ok && m != "" {
						participant = m
					} else {
						participant = sender
					}
				}

				// Try Evolution API format
				if dataObj, ok := raw["data"].(map[string]interface{}); ok {
					if keyObj, ok := dataObj["key"].(map[string]interface{}); ok {
						if remoteJid, ok := keyObj["remoteJid"].(string); ok {
							sender = strings.Split(remoteJid, "@")[0]
							fromJid = remoteJid
						}
					}
					if msgObj, ok := dataObj["message"].(map[string]interface{}); ok {
						if conv, ok := msgObj["conversation"].(string); ok && conv != "" {
							message = conv
						} else if extObj, ok := msgObj["extendedTextMessage"].(map[string]interface{}); ok {
							if text, ok := extObj["text"].(string); ok {
								message = text
							}
						} else if imgMsg, ok := msgObj["imageMessage"].(map[string]interface{}); ok {
							if capText, ok := imgMsg["caption"].(string); ok && capText != "" {
								message = capText
							}
							if urlStr, ok := imgMsg["mediaUrl"].(string); ok && urlStr != "" {
								mediaURL = urlStr
							} else if urlStr, ok := imgMsg["url"].(string); ok && urlStr != "" {
								mediaURL = urlStr
							}
						}
					}
					if part, ok := dataObj["participant"].(string); ok && part != "" {
						participant = strings.Split(part, "@")[0]
					}
				}

				// Try Waha format
				if payload, ok := raw["payload"].(map[string]interface{}); ok {
					// Check and ignore messages sent by the bot itself (fromMe)
					if fromMe, ok := payload["fromMe"].(bool); ok && fromMe {
						logJSON("info", "self_sent_ignored", map[string]any{"session": raw["session"]})
						w.WriteHeader(http.StatusOK)
						return
					}

					if from, ok := payload["from"].(string); ok {
						sender = strings.Split(from, "@")[0]
						fromJid = from
					}
					if body, ok := payload["body"].(string); ok {
						message = body
					}
					if capStr, ok := payload["caption"].(string); ok && capStr != "" {
						message = capStr
					}
					if mediaObj, ok := payload["media"].(map[string]interface{}); ok {
						if urlStr, ok := mediaObj["url"].(string); ok && urlStr != "" {
							mediaURL = urlStr
						}
					}
					if urlStr, ok := payload["mediaUrl"].(string); ok && urlStr != "" {
						mediaURL = urlStr
					}
					if part, ok := payload["participant"].(string); ok && part != "" {
						participant = strings.Split(part, "@")[0]
					}
				}

				// Try Official WhatsApp Cloud API format
				if entryObj, ok := raw["entry"].([]interface{}); ok && len(entryObj) > 0 {
					if entryMap, ok := entryObj[0].(map[string]interface{}); ok {
						if changesObj, ok := entryMap["changes"].([]interface{}); ok && len(changesObj) > 0 {
							if changesMap, ok := changesObj[0].(map[string]interface{}); ok {
								if valueMap, ok := changesMap["value"].(map[string]interface{}); ok {
									if messagesObj, ok := valueMap["messages"].([]interface{}); ok && len(messagesObj) > 0 {
										if msgMap, ok := messagesObj[0].(map[string]interface{}); ok {
											if from, ok := msgMap["from"].(string); ok {
												sender = from
											}
											if textMap, ok := msgMap["text"].(map[string]interface{}); ok {
												if body, ok := textMap["body"].(string); ok {
													message = body
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	sender = strings.TrimSpace(sender)
	message = strings.TrimSpace(message)
	mediaURL = strings.TrimSpace(mediaURL)

	if message == "" && mediaURL != "" {
		message = "!scan"
	}

	if mediaURL != "" && strings.HasPrefix(strings.ToLower(message), "!scan") {
		message = message + " " + mediaURL
	}

	if sender == "" || message == "" {
		logJSON("warn", "unrecognized_payload", map[string]any{"sender": sender, "message": message})
		w.WriteHeader(http.StatusOK)
		return
	}

	// Clean up sender format (e.g., remove "+" from "+62812...")
	sender = strings.ReplaceAll(sender, "+", "")
	participant = strings.ReplaceAll(participant, "+", "")

	isGroup := false
	if fromJid != "" && (strings.Contains(fromJid, "@g.us") || strings.Contains(fromJid, "@us")) {
		isGroup = true
	}

	if isGroup {
		// Extract dynamic bot info from webhook root 'me' if available
		var meId, meName string
		if raw != nil {
			if meObj, ok := raw["me"].(map[string]interface{}); ok {
				if id, ok := meObj["id"].(string); ok {
					meId = strings.Split(id, "@")[0]
				}
				if name, ok := meObj["pushName"].(string); ok {
					meName = name
				}
			}
		}

		botPhone := h.service.GetWhatsAppBotNumber(r.Context())
		botPhoneClean := strings.ReplaceAll(botPhone, "+", "")
		if botPhoneClean == "" {
			botPhoneClean = meId
		}

		isMentioned := false
		if botPhoneClean != "" && strings.Contains(message, "@"+botPhoneClean) {
			isMentioned = true
		}
		if botPhoneClean != "" && strings.Contains(message, botPhoneClean) && strings.Contains(message, "@") {
			isMentioned = true
		}

		// Fallback check by bot's pushName (e.g., "Aish | Support HONET")
		lowerMsg := strings.ToLower(message)
		if meName != "" {
			cleanMeName := strings.ToLower(strings.TrimSpace(meName))
			if strings.Contains(lowerMsg, "@"+cleanMeName) || strings.Contains(lowerMsg, "@~"+cleanMeName) {
				isMentioned = true
			}
			parts := strings.Split(cleanMeName, "|")
			for _, part := range parts {
				trimmedPart := strings.TrimSpace(part)
				if trimmedPart != "" && (strings.Contains(lowerMsg, "@"+trimmedPart) || strings.Contains(lowerMsg, "@~"+trimmedPart)) {
					isMentioned = true
				}
			}
		}

		if strings.Contains(lowerMsg, "@pekan") || strings.Contains(lowerMsg, "@bot") {
			isMentioned = true
		}

		if !isMentioned {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Clean mentions from message
		if botPhoneClean != "" {
			message = strings.ReplaceAll(message, "@"+botPhoneClean, "")
			message = strings.ReplaceAll(message, botPhoneClean, "")
		}
		if meName != "" {
			message = strings.ReplaceAll(message, "@"+meName, "")
			message = strings.ReplaceAll(message, "@~"+meName, "")
			parts := strings.Split(meName, "|")
			for _, part := range parts {
				trimmedPart := strings.TrimSpace(part)
				if trimmedPart != "" {
					message = strings.ReplaceAll(message, "@"+trimmedPart, "")
					message = strings.ReplaceAll(message, "@~"+trimmedPart, "")
				}
			}
		}
		message = strings.ReplaceAll(message, "@pekan", "")
		message = strings.ReplaceAll(message, "@PEKAN", "")
		message = strings.ReplaceAll(message, "@bot", "")
		message = strings.ReplaceAll(message, "@BOT", "")
		message = strings.TrimSpace(message)

		if participant != "" {
			sender = participant
		}
	}

	recipient := sender
	if fromJid != "" {
		recipient = fromJid
	}

	// 1. Intercept login commands or raw OTP codes
	cleanMsg := strings.ToLower(strings.TrimSpace(message))
	isOtpLogin := false
	var loginCode string

	if strings.HasPrefix(cleanMsg, "!login ") {
		parts := strings.SplitN(message, " ", 2)
		if len(parts) == 2 {
			loginCode = strings.ToUpper(strings.TrimSpace(parts[1]))
			isOtpLogin = true
		}
	} else if strings.HasPrefix(cleanMsg, "wa-") && len(cleanMsg) == 9 {
		loginCode = strings.ToUpper(strings.TrimSpace(message))
		isOtpLogin = true
	}

	if isOtpLogin {
		err := h.service.ProcessLogin(r.Context(), sender, loginCode)
		if err != nil {
			logJSON("error", "login_failed", map[string]any{"sender": sender, "error": err.Error()})
			
			reply := "Gagal Menghubungkan Akun\n\nKode OTP tidak valid atau sudah kadaluwarsa. Silakan coba generate kode baru dari dasbor PEKAN."
			if strings.Contains(strings.ToLower(err.Error()), "nomor") {
				reply = "Gagal Menghubungkan Akun\n\n" + err.Error() + "."
			}

			_ = h.service.SendWhatsAppMessage(r.Context(), recipient, reply)
		} else {
			logJSON("info", "login_success", map[string]any{"sender": sender})
			
			reply := "Akun Berhasil Terhubung!\n\nNomor WhatsApp Anda telah berhasil dihubungkan ke akun PEKAN.\nSekarang Anda dapat mencatat transaksi secara otomatis menggunakan asisten AI PEKAN."
			_ = h.service.SendWhatsAppMessage(r.Context(), recipient, reply)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	// 2. Not a login command. Push to asynchronous database queue.
	var tenantID *string
	var userID *string
	sess, sErr := h.service.GetSessionByPhone(r.Context(), sender)
	if sErr == nil {
		tID := sess.TenantID
		uID := sess.UserID
		tenantID = &tID
		userID = &uID
	}

	_, qErr := h.service.EnqueueMessage(r.Context(), recipient, message, tenantID, userID)
	if qErr != nil {
		logJSON("error", "enqueue_failed", map[string]any{"sender": sender, "error": qErr.Error()})
		// Fallback to sync processing if database enqueue fails
		var tIDVal, uIDVal string
		if tenantID != nil {
			tIDVal = *tenantID
		}
		if userID != nil {
			uIDVal = *userID
		}
		replyText, err := h.service.ProcessAIChat(r.Context(), sender, message, tIDVal, uIDVal)
		if err != nil {
			logJSON("error", "ai_chat_fallback_error", map[string]any{"sender": sender, "error": err.Error()})
		}
		if replyText != "" {
			_ = h.service.SendWhatsAppMessage(r.Context(), recipient, replyText)
		}
	}

	w.WriteHeader(http.StatusOK)
}
