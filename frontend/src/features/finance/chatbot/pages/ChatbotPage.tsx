import { useEffect, useState, useRef } from "react";
import { useParams, Link } from "react-router-dom";
import { useI18n } from "../../../../core/i18n/i18n";
import { checkWhatsAppStatus, sendChatbotMessage, WhatsAppStatus } from "../../settings/api/whatsapp.api";
import { useToast } from "../../../../core/hooks/useToast";
import { ToastContainer } from "../../../../core/components/Toast";
import { PageHeader } from "../../../../core/components/PageHeader";

type ChatMessage = {
  id: string;
  sender: "user" | "bot";
  text: string;
  timestamp: Date;
};

export function ChatbotPage(): JSX.Element {
  const { tenantCode } = useParams<{ tenantCode: string }>();
  const { t } = useI18n();
  const { toasts, success, error: showError, remove: removeToast } = useToast();

  const [activeTab, setActiveTab] = useState<"web" | "whatsapp">("web");
  const [waStatus, setWaStatus] = useState<WhatsAppStatus | null>(null);
  const [loadingStatus, setLoadingStatus] = useState(true);
  const [inputMessage, setInputMessage] = useState("");
  const [chatHistory, setChatHistory] = useState<ChatMessage[]>([]);
  const [sending, setSending] = useState(false);

  const chatEndRef = useRef<HTMLDivElement>(null);

  // Initialize with welcome message
  useEffect(() => {
    setChatHistory([
      {
        id: "welcome",
        sender: "bot",
        text: "👋 **Halo! Saya Asisten AI Pekan.**\n\nSaya siap membantu Anda memantau keuangan, menganalisis pengeluaran, mengecek anggaran, dan memberikan tips finansial terbaik secara langsung.\n\n*Silakan tanyakan apa saja mengenai catatan keuangan Anda!*",
        timestamp: new Date()
      }
    ]);
    loadWhatsAppStatus();
  }, []);

  // Auto-scroll to bottom of chat
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [chatHistory, sending]);

  async function loadWhatsAppStatus() {
    setLoadingStatus(true);
    try {
      const status = await checkWhatsAppStatus();
      setWaStatus(status);
    } catch (err) {
      console.error("Failed to load WhatsApp Status", err);
    } finally {
      setLoadingStatus(false);
    }
  }

  // Simple, robust Markdown parser for standard bot replies
  function renderMarkdown(text: string) {
    const lines = text.split("\n");
    return lines.map((line, idx) => {
      // Bullets check
      let isBullet = false;
      let cleanLine = line.trim();
      if (cleanLine.startsWith("- ") || cleanLine.startsWith("* ")) {
        isBullet = true;
        cleanLine = cleanLine.substring(2);
      }

      // Check bold syntax: **text** or *text*
      const boldRegex = /\*\*([^*]+)\*\*|\*([^*]+)\*/g;
      const parts: JSX.Element[] = [];
      let lastIndex = 0;
      let match;

      while ((match = boldRegex.exec(cleanLine)) !== null) {
        const matchStr = match[0];
        const content = match[1] || match[2];
        const matchIndex = match.index;

        if (matchIndex > lastIndex) {
          parts.push(<span key={`text-${lastIndex}`}>{cleanLine.substring(lastIndex, matchIndex)}</span>);
        }

        parts.push(<strong key={`bold-${matchIndex}`} style={{ fontWeight: 700, color: "inherit" }}>{content}</strong>);
        lastIndex = boldRegex.lastIndex;
      }

      if (lastIndex < cleanLine.length) {
        parts.push(<span key={`text-${lastIndex}`}>{cleanLine.substring(lastIndex)}</span>);
      }

      const contentElement = parts.length > 0 ? parts : cleanLine;

      if (isBullet) {
        return (
          <li key={idx} style={{ marginLeft: "1.25rem", marginBottom: "0.25rem", listStyleType: "disc" }}>
            {contentElement}
          </li>
        );
      }

      return (
        <p key={idx} style={{ margin: "0 0 0.5rem 0", minHeight: cleanLine === "" ? "0.75rem" : "auto", lineHeight: "1.6" }}>
          {contentElement}
        </p>
      );
    });
  }

  async function handleSendMessage(messageText: string = inputMessage) {
    const trimmed = messageText.trim();
    if (!trimmed || sending) return;

    // Add user message to chat
    const userMsgId = `user-${Date.now()}`;
    const newHistory = [...chatHistory, {
      id: userMsgId,
      sender: "user" as const,
      text: trimmed,
      timestamp: new Date()
    }];
    setChatHistory(newHistory);
    setInputMessage("");
    setSending(true);

    try {
      const reply = await sendChatbotMessage(trimmed);
      setChatHistory([
        ...newHistory,
        {
          id: `bot-${Date.now()}`,
          sender: "bot" as const,
          text: reply || "Maaf, Asisten AI sedang mengalami kendala. Silakan coba kembali.",
          timestamp: new Date()
        }
      ]);
    } catch (err: any) {
      showError(err.message || "Gagal memproses jawaban Asisten AI.");
      setChatHistory([
        ...newHistory,
        {
          id: `bot-err-${Date.now()}`,
          sender: "bot" as const,
          text: "⚠️ **Kesalahan Sistem**\n\nGagal terhubung dengan server AI. Harap pastikan kunci API di pengaturan **Scan Struk** Anda sudah terkonfigurasi dengan benar.",
          timestamp: new Date()
        }
      ]);
    } finally {
      setSending(false);
    }
  }

  function handleClearChat() {
    if (window.confirm("Apakah Anda ingin membersihkan riwayat percakapan ini?")) {
      setChatHistory([
        {
          id: "welcome",
          sender: "bot",
          text: "👋 **Halo! Saya Asisten AI Pekan.**\n\nSaya siap membantu Anda memantau keuangan, menganalisis pengeluaran, mengecek anggaran, dan memberikan tips finansial terbaik secara langsung.\n\n*Silakan tanyakan apa saja mengenai catatan keuangan Anda!*",
          timestamp: new Date()
        }
      ]);
      success("Percakapan telah dibersihkan.");
    }
  }

  function handleCopyChat() {
    const textToCopy = chatHistory
      .map(m => `[${m.sender === "bot" ? "AI" : "Anda"}]: ${m.text}`)
      .join("\n\n");
    navigator.clipboard.writeText(textToCopy);
    success("Seluruh percakapan disalin!");
  }

  const suggestedPrompts = [
    "Berapa pengeluaran saya bulan ini?",
    "Tampilkan ringkasan anggaran saya",
    "Beri saya tips menghemat uang",
    "Tampilkan 3 transaksi terakhir"
  ];

  return (
    <section className="page-section" style={{ maxWidth: "1000px", margin: "0 auto" }}>
      <PageHeader 
        title="Asisten AI Pekan" 
        description="Portal konsultasi keuangan cerdas Anda. Pilih untuk berinteraksi langsung di aplikasi ini atau integrasikan dengan nomor WhatsApp Anda." 
      />

      {/* Tabs */}
      <div style={{
        display: "flex",
        background: "var(--surface-soft)",
        padding: "4px",
        borderRadius: "10px",
        marginTop: "1.5rem",
        marginBottom: "1.5rem",
        border: "1px solid var(--border)",
        maxWidth: "fit-content"
      }}>
        <button
          onClick={() => setActiveTab("web")}
          style={{
            padding: "8px 20px",
            borderRadius: "8px",
            border: "none",
            background: activeTab === "web" ? "var(--primary)" : "transparent",
            color: activeTab === "web" ? "#fff" : "var(--text-muted)",
            fontWeight: 600,
            cursor: "pointer",
            transition: "all 0.2s"
          }}
        >
          Chat Web UI
        </button>
        <button
          onClick={() => setActiveTab("whatsapp")}
          style={{
            padding: "8px 20px",
            borderRadius: "8px",
            border: "none",
            background: activeTab === "whatsapp" ? "var(--primary)" : "transparent",
            color: activeTab === "whatsapp" ? "#fff" : "var(--text-muted)",
            fontWeight: 600,
            cursor: "pointer",
            transition: "all 0.2s"
          }}
        >
          Integrasi WhatsApp
        </button>
      </div>

      {activeTab === "web" ? (
        <div className="card surface" style={{ display: "flex", flexDirection: "column", height: "650px", padding: 0, overflow: "hidden", border: "1px solid var(--border)" }}>
          {/* Header Panel */}
          <div style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "1rem 1.5rem",
            borderBottom: "1px solid var(--border)",
            background: "rgba(255, 255, 255, 0.02)"
          }}>
            <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
              <div style={{
                width: "10px",
                height: "10px",
                borderRadius: "50%",
                background: sending ? "var(--warning)" : "#10b981",
                boxShadow: sending ? "0 0 8px var(--warning)" : "0 0 8px #10b981"
              }} />
              <div>
                <h4 style={{ margin: 0, fontWeight: 600 }}>Pekan AI Engine</h4>
                <p style={{ margin: 0, fontSize: "0.75rem", color: "var(--muted)" }}>
                  {sending ? "Sedang memproses..." : "Aktif & Siap Membantu"}
                </p>
              </div>
            </div>

            <div style={{ display: "flex", gap: "8px" }}>
              <button 
                type="button" 
                onClick={handleCopyChat}
                style={{
                  background: "transparent",
                  border: "none",
                  color: "var(--muted)",
                  fontSize: "0.85rem",
                  cursor: "pointer",
                  padding: "4px 8px",
                  borderRadius: "4px"
                }}
                title="Salin Percakapan"
              >
                Salin
              </button>
              <button 
                type="button" 
                onClick={handleClearChat}
                style={{
                  background: "transparent",
                  border: "none",
                  color: "var(--muted)",
                  fontSize: "0.85rem",
                  cursor: "pointer",
                  padding: "4px 8px",
                  borderRadius: "4px"
                }}
                title="Bersihkan Percakapan"
              >
                Reset
              </button>
            </div>
          </div>

          {/* Messages Body */}
          <div style={{
            flex: 1,
            overflowY: "auto",
            padding: "1.5rem",
            background: "rgba(0, 0, 0, 0.05)",
            display: "flex",
            flexDirection: "column",
            gap: "1.25rem"
          }}>
            {chatHistory.map((msg) => {
              const isBot = msg.sender === "bot";
              return (
                <div 
                  key={msg.id} 
                  style={{
                    display: "flex",
                    justifyContent: isBot ? "flex-start" : "flex-end",
                    width: "100%"
                  }}
                >
                  <div style={{
                    maxWidth: "80%",
                    display: "flex",
                    flexDirection: "column",
                    alignItems: isBot ? "flex-start" : "flex-end"
                  }}>
                    {/* Bubble */}
                    <div style={{
                      padding: "12px 16px",
                      borderRadius: isBot ? "16px 16px 16px 2px" : "16px 16px 2px 16px",
                      background: isBot ? "var(--surface-soft)" : "linear-gradient(135deg, var(--primary), var(--primary-dark))",
                      color: isBot ? "var(--text)" : "#fff",
                      border: isBot ? "1px solid var(--border)" : "none",
                      boxShadow: "0 2px 8px rgba(0,0,0,0.05)",
                      fontSize: "0.95rem"
                    }}>
                      {renderMarkdown(msg.text)}
                    </div>
                    {/* Timestamp */}
                    <span style={{
                      fontSize: "0.75rem",
                      color: "var(--muted)",
                      marginTop: "4px",
                      padding: "0 4px"
                    }}>
                      {msg.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </span>
                  </div>
                </div>
              );
            })}

            {sending && (
              <div style={{ display: "flex", justifyContent: "flex-start", width: "100%" }}>
                <div style={{
                  background: "var(--surface-soft)",
                  border: "1px solid var(--border)",
                  padding: "12px 20px",
                  borderRadius: "16px 16px 16px 2px",
                  display: "flex",
                  alignItems: "center",
                  gap: "6px"
                }}>
                  <span style={{ fontSize: "0.85rem", color: "var(--muted)", fontStyle: "italic" }}>
                    Asisten AI sedang berpikir
                  </span>
                  <div style={{ display: "flex", gap: "3px" }}>
                    <div className="dot-bounce" style={{ width: "4px", height: "4px", background: "var(--primary)", borderRadius: "50%", animation: "bounce 1.4s infinite ease-in-out both" }} />
                    <div className="dot-bounce" style={{ width: "4px", height: "4px", background: "var(--primary)", borderRadius: "50%", animation: "bounce 1.4s infinite ease-in-out both 0.2s" }} />
                    <div className="dot-bounce" style={{ width: "4px", height: "4px", background: "var(--primary)", borderRadius: "50%", animation: "bounce 1.4s infinite ease-in-out both 0.4s" }} />
                  </div>
                </div>
              </div>
            )}

            <div ref={chatEndRef} />
          </div>

          {/* Quick Prompts */}
          {chatHistory.length <= 1 && !sending && (
            <div style={{
              padding: "0.75rem 1.5rem",
              background: "rgba(0, 0, 0, 0.08)",
              borderTop: "1px solid var(--border)",
              display: "flex",
              flexWrap: "wrap",
              gap: "8px"
            }}>
              {suggestedPrompts.map((prompt, idx) => (
                <button
                  key={idx}
                  onClick={() => handleSendMessage(prompt)}
                  style={{
                    background: "var(--surface-soft)",
                    border: "1px solid var(--border)",
                    color: "var(--text-muted)",
                    padding: "6px 14px",
                    borderRadius: "20px",
                    fontSize: "0.82rem",
                    fontWeight: 500,
                    cursor: "pointer",
                    transition: "all 0.2s"
                  }}
                  onMouseOver={(e) => {
                    e.currentTarget.style.borderColor = "var(--primary)";
                    e.currentTarget.style.color = "var(--primary)";
                  }}
                  onMouseOut={(e) => {
                    e.currentTarget.style.borderColor = "var(--border)";
                    e.currentTarget.style.color = "var(--text-muted)";
                  }}
                >
                  {prompt}
                </button>
              ))}
            </div>
          )}

          {/* Form Action */}
          <form 
            onSubmit={(e) => { e.preventDefault(); handleSendMessage(); }}
            style={{
              padding: "1rem 1.5rem",
              borderTop: "1px solid var(--border)",
              background: "rgba(255, 255, 255, 0.02)",
              display: "flex",
              gap: "12px"
            }}
          >
            <input
              type="text"
              value={inputMessage}
              onChange={(e) => setInputMessage(e.target.value)}
              placeholder="Tanyakan analisis finansial Anda..."
              disabled={sending}
              style={{
                flex: 1,
                background: "var(--surface-soft)",
                border: "1px solid var(--border)",
                borderRadius: "8px",
                padding: "10px 14px",
                color: "var(--text)",
                fontSize: "0.95rem"
              }}
            />
            <button
              type="submit"
              disabled={sending || !inputMessage.trim()}
              className="btn btn-primary"
              style={{
                padding: "0 24px",
                borderRadius: "8px",
                height: "42px",
                fontWeight: 600
              }}
            >
              Kirim
            </button>
          </form>
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: "1.5rem" }}>
          {/* Status WhatsApp Card */}
          <div className="card surface">
            <h3 style={{ display: "flex", alignItems: "center", gap: "8px", marginTop: 0 }}>
              <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.888-.788-1.489-1.761-1.663-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413Z"/>
              </svg>
              Integrasi WhatsApp Bot AI
            </h3>

            {loadingStatus ? (
              <p>Memeriksa status integrasi...</p>
            ) : waStatus?.connected ? (
              <div style={{ background: "rgba(16, 185, 129, 0.1)", border: "1px solid rgba(16, 185, 129, 0.2)", borderRadius: "8px", padding: "1.5rem" }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '1rem' }}>
                  <div style={{ width: '12px', height: '12px', borderRadius: '50%', background: '#10b981', boxShadow: '0 0 0 4px rgba(16,185,129,0.2)' }} />
                  <h4 style={{ margin: 0, color: '#10b981' }}>Terhubung Aktif</h4>
                </div>
                <p style={{ margin: "0 0 1.25rem 0", color: "var(--text-muted)", fontSize: "0.95rem" }}>
                  Nomor WhatsApp Anda <strong>{waStatus.phone_number}</strong> telah berhasil ditautkan. Anda dapat langsung bercakap-cakap dengan Bot Keuangan kami lewat WhatsApp.
                </p>
                {waStatus.bot_phone_number ? (
                  <a 
                    href={`https://wa.me/${waStatus.bot_phone_number.replace(/\+/g, "")}?text=Halo%20Pekan`}
                    target="_blank" 
                    rel="noopener noreferrer" 
                    className="btn btn-primary"
                    style={{
                      display: "inline-flex",
                      alignItems: "center",
                      gap: "8px",
                      padding: "10px 20px",
                      fontWeight: 600,
                      textDecoration: "none",
                      background: "#25D366",
                      borderColor: "#25D366",
                      color: "#fff"
                    }}
                  >
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                      <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.888-.788-1.489-1.761-1.663-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413Z"/>
                    </svg>
                    Buka Chat WhatsApp Bot ({waStatus.bot_phone_number})
                  </a>
                ) : (
                  <p style={{ color: "var(--muted)", fontStyle: "italic", fontSize: "0.9rem" }}>
                    Silakan hubungi WhatsApp Bot sistem Anda langsung dari HP Anda.
                  </p>
                )}
              </div>
            ) : (
              <div style={{ background: "var(--surface-soft)", border: "1px solid var(--border)", borderRadius: "8px", padding: "1.5rem" }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '1rem' }}>
                  <div style={{ width: '12px', height: '12px', borderRadius: '50%', background: 'var(--danger)', boxShadow: '0 0 0 4px rgba(239,68,68,0.2)' }} />
                  <h4 style={{ margin: 0, color: 'var(--danger)' }}>Belum Terhubung</h4>
                </div>
                <p style={{ margin: "0 0 1.25rem 0", color: "var(--text-muted)", fontSize: "0.95rem", lineHeight: "1.6" }}>
                  Anda belum mengintegrasikan nomor WhatsApp dengan workspace ini. Dengan menghubungkannya, Anda bisa mencatat transaksi cukup dengan mengetik pesan biasa di WhatsApp.
                </p>
                <Link 
                  to={`/app/${tenantCode}/finance/settings/whatsapp`} 
                  className="btn btn-primary"
                  style={{
                    display: "inline-flex",
                    alignItems: "center",
                    padding: "10px 20px",
                    fontWeight: 600,
                    textDecoration: "none"
                  }}
                >
                  Tautkan Nomor Sekarang
                </Link>
              </div>
            )}
          </div>

          {/* WhatsApp Instructions Card */}
          <div className="card surface">
            <h4 style={{ marginTop: 0 }}>Cara Penggunaan Bot WhatsApp Pekan</h4>
            <div style={{ display: "flex", flexDirection: "column", gap: "1rem", fontSize: "0.92rem", color: "var(--text-muted)", lineHeight: "1.6" }}>
              <div style={{ display: "flex", gap: "12px" }}>
                <div style={{ width: "24px", height: "24px", borderRadius: "50%", background: "var(--primary)", color: "#fff", display: "flex", alignItems: "center", justifyItems: "center", justifyContent: "center", fontWeight: 700, flexShrink: 0 }}>1</div>
                <div>
                  <strong>Mencatat Transaksi Otomatis</strong>
                  <br />Kirim pesan alami ke bot WhatsApp, misalnya: <em>"Beli kopi 25rb kemarin sore"</em> atau <em>"Gaji masuk 5 juta"</em>. Bot AI akan memproses dan mengonfirmasi pencatatan Anda.
                </div>
              </div>
              <div style={{ display: "flex", gap: "12px" }}>
                <div style={{ width: "24px", height: "24px", borderRadius: "50%", background: "var(--primary)", color: "#fff", display: "flex", alignItems: "center", justifyItems: "center", justifyContent: "center", fontWeight: 700, flexShrink: 0 }}>2</div>
                <div>
                  <strong>Mengelola Transaksi</strong>
                  <br />Gunakan perintah untuk merubah transaksi dengan menyebutkan ID-nya: <em>"ubah transaksi c5e8211b nominalnya jadi 75rb"</em> atau <em>"hapus transaksi c5e8211b"</em>.
                </div>
              </div>
              <div style={{ display: "flex", gap: "12px" }}>
                <div style={{ width: "24px", height: "24px", borderRadius: "50%", background: "var(--primary)", color: "#fff", display: "flex", alignItems: "center", justifyItems: "center", justifyContent: "center", fontWeight: 700, flexShrink: 0 }}>3</div>
                <div>
                  <strong>Menganalisis Keuangan</strong>
                  <br />Tanyakan laporan keuangan Anda secara fleksibel: <em>"Berapa sisa anggaran makan saya?"</em> atau <em>"Berapa pengeluaran saya minggu ini?"</em>.
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Bounce keyframe styling */}
      <style>{`
        @keyframes bounce {
          0%, 80%, 100% { transform: scale(0); }
          40% { transform: scale(1.0); }
        }
      `}</style>

      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </section>
  );
}
