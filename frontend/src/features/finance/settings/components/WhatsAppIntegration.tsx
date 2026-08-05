import { useEffect, useState, useRef } from "react";
import { useNavigate, useParams, Link } from "react-router-dom";
import { checkWhatsAppStatus, disconnectWhatsApp, connectWhatsApp, WhatsAppStatus, generateWhatsAppOTP } from "../api/whatsapp.api";
import { getMeProfile, MeProfileResponse } from "../../../../core/auth/auth-api";
import { useToast } from "../../../../core/hooks/useToast";
import { ToastContainer } from "../../../../core/components/Toast";

export function WhatsAppIntegration(): JSX.Element {
  const { tenantCode } = useParams<{ tenantCode: string }>();
  const { toasts, success, error: showToastError, remove: removeToast } = useToast();
  const navigate = useNavigate();
  const [status, setStatus] = useState<WhatsAppStatus | null>(null);
  const [profile, setProfile] = useState<MeProfileResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // New OTP UI State
  const [otpCode, setOtpCode] = useState<string | null>(null);
  const [generatingOtp, setGeneratingOtp] = useState(false);

  const pollIntervalRef = useRef<any>(null);

  useEffect(() => {
    loadStatus();
    return () => {
      stopPolling();
    };
  }, []);

  // When an OTP is active, poll the status every 5 seconds so the UI automatically
  // connects when the user chats "!login <OTP>" on WhatsApp.
  useEffect(() => {
    if (otpCode && !status?.connected) {
      startPolling();
    } else {
      stopPolling();
    }
  }, [otpCode, status?.connected]);

  function startPolling() {
    stopPolling();
    pollIntervalRef.current = setInterval(async () => {
      try {
        const statusData = await checkWhatsAppStatus();
        if (statusData?.connected) {
          setStatus(statusData);
          success("WhatsApp berhasil terhubung!");
          setOtpCode(null);
          stopPolling();
        }
      } catch (err) {
        // Silently fail polling
      }
    }, 5000);
  }

  function stopPolling() {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
  }

  async function loadStatus() {
    setLoading(true);
    setError(null);
    try {
      const [statusData, profileData] = await Promise.all([
        checkWhatsAppStatus(),
        getMeProfile()
      ]);
      setStatus(statusData);
      setProfile(profileData);
    } catch (err: any) {
      console.error("[WhatsApp] Failed to load data:", err);
      setError("Gagal memuat status integrasi WhatsApp.");
    } finally {
      setLoading(false);
    }
  }


  async function handleGenerateOtp() {
    setGeneratingOtp(true);
    setError(null);
    try {
      const code = await generateWhatsAppOTP();
      if (code) {
        setOtpCode(code);
        success("Kode OTP berhasil digenerate!");
      } else {
        throw new Error("Gagal menerima kode OTP dari server.");
      }
    } catch (err: any) {
      console.error("[WhatsApp] Failed to generate OTP:", err);
      const msg = err.message || "Gagal membuat kode OTP.";
      setError(msg);
      showToastError(msg);
    } finally {
      setGeneratingOtp(false);
    }
  }

  function handleCopy(text: string, msg: string = "Teks berhasil disalin!") {
    navigator.clipboard.writeText(text);
    success(msg);
  }

  async function handleDisconnect() {
    if (!window.confirm("Apakah Anda yakin ingin memutuskan integrasi WhatsApp?")) return;
    setLoading(true);
    try {
      await disconnectWhatsApp();
      success("Koneksi WhatsApp diputuskan.");
      setOtpCode(null);
      await loadStatus();
    } catch (err: any) {
      console.error("[WhatsApp] Failed to disconnect:", err);
      const msg = "Gagal memutuskan koneksi WhatsApp.";
      setError(msg);
      showToastError(msg);
      setLoading(false);
    }
  }

  if (loading && !status) {
    return <div className="card surface"><p>Memuat integrasi WhatsApp...</p></div>;
  }

  return (
    <div className="card surface" style={{ position: 'relative' }}>
      <h3 className="form-title" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
          <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.888-.788-1.489-1.761-1.663-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413Z"/>
        </svg>
        Integrasi WhatsApp Bot AI
      </h3>
      <p className="page-subtitle" style={{ marginBottom: "1.5rem" }}>
        Hubungkan nomor WhatsApp Anda untuk mencatat pengeluaran, melihat sisa anggaran, dan membuat ringkasan keuangan semudah ber-chatting.
      </p>

      {error && <p className="alert error">{error}</p>}

      {status?.connected ? (
        <div style={{ background: "rgba(16, 185, 129, 0.1)", border: "1px solid rgba(16, 185, 129, 0.2)", borderRadius: "8px", padding: "1.5rem" }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '1rem' }}>
            <div style={{ width: '12px', height: '12px', borderRadius: '50%', background: '#10b981', boxShadow: '0 0 0 4px rgba(16,185,129,0.2)' }} />
            <h4 style={{ margin: 0, color: '#10b981' }}>Terhubung Aktif</h4>
          </div>
          <div className="form-grid">
            <div>
              <p style={{ margin: 0, fontSize: '0.85rem', color: 'var(--muted)' }}>Nomor WhatsApp</p>
              <p style={{ margin: '4px 0 0 0', fontWeight: 600 }}>{status.phone_number}</p>
            </div>
            <div>
              <p style={{ margin: 0, fontSize: '0.85rem', color: 'var(--muted)' }}>Terakhir Aktif</p>
              <p style={{ margin: '4px 0 0 0', fontWeight: 600 }}>{status.last_active ? new Date(status.last_active).toLocaleString('id-ID') : 'Belum ada aktivitas'}</p>
            </div>
          </div>
          <div style={{ marginTop: '1.5rem', display: 'flex', gap: '1rem' }}>
            <button className="btn btn-ghost danger" onClick={handleDisconnect} disabled={loading}>
              Putuskan Koneksi
            </button>
          </div>
        </div>
      ) : (
        <div style={{ background: "var(--surface-soft)", border: "1px solid var(--border)", borderRadius: "12px", padding: "1.5rem", textAlign: "left" }}>
          <p style={{ marginBottom: '1.25rem', color: 'var(--text)', fontSize: '1rem' }}>
            <strong>Hubungkan Nomor WhatsApp Anda</strong>
          </p>

          <p style={{ margin: '0 0 1.25rem 0', color: 'var(--muted)', fontSize: '0.9rem', lineHeight: '1.5' }}>
            Generate kode OTP Anda di bawah ini, kemudian kirimkan pesan aktivasi ke WhatsApp Bot untuk menghubungkan nomor Anda secara aman.
          </p>

          {!otpCode ? (
            <button 
              onClick={handleGenerateOtp}
              className="btn btn-primary"
              disabled={generatingOtp}
              style={{
                padding: '12px 24px',
                fontSize: '15px',
                fontWeight: 600,
                height: '46px'
              }}
            >
              {generatingOtp ? "Sedang Membuat..." : "Generate Kode OTP"}
            </button>
          ) : (
            <div style={{ 
              background: 'var(--surface)', 
              border: '1px solid var(--border)', 
              borderRadius: '10px', 
              padding: '1.5rem', 
              boxShadow: '0 4px 12px rgba(0,0,0,0.05)'
            }}>
              <div style={{ textAlign: 'center', marginBottom: '1.5rem' }}>
                <p style={{ margin: '0 0 0.5rem 0', fontSize: '0.85rem', color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '1px' }}>
                  Kirim Pesan Berikut ke WhatsApp (Berlaku 10 Menit)
                </p>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem', alignItems: 'center' }}>
                  <div style={{ display: 'inline-flex', alignItems: 'center', gap: '1.25rem', background: 'var(--surface-soft)', padding: '12px 24px', borderRadius: '8px', border: '1px solid var(--border)' }}>
                    <span style={{ fontSize: '1.75rem', fontWeight: 800, fontFamily: 'monospace', color: 'var(--primary)' }}>
                      !login {otpCode}
                    </span>
                    <button 
                      type="button" 
                      onClick={() => handleCopy(`!login ${otpCode}`, "Perintah login berhasil disalin!")}
                      style={{
                        background: 'rgba(16, 185, 129, 0.1)',
                        border: 'none',
                        color: 'var(--primary)',
                        borderRadius: '4px',
                        padding: '6px 12px',
                        fontSize: '0.85rem',
                        fontWeight: 600,
                        cursor: 'pointer',
                        transition: 'all 0.2s'
                      }}
                    >
                      Salin Perintah
                    </button>
                  </div>

                  <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap', justifyContent: 'center' }}>
                    <a 
                      href={`https://wa.me/${status?.bot_phone_number ? status.bot_phone_number.replace(/\+/g, "") : ""}?text=!login%20${otpCode}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="btn btn-primary"
                      style={{ background: '#25D366', borderColor: '#25D366', color: '#fff', display: 'inline-flex', alignItems: 'center', gap: '8px' }}
                    >
                      <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                        <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.888-.788-1.489-1.761-1.663-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413Z"/>
                      </svg>
                      Direct WhatsApp Bot
                    </a>
                    <Link 
                      to={`/app/${tenantCode}/finance/chatbot`} 
                      className="btn btn-secondary-outline"
                      style={{ display: 'inline-flex', alignItems: 'center', gap: '8px' }}
                    >
                      Buka WebUI Chat Bot
                    </Link>
                  </div>
                  {!status?.bot_phone_number && (
                    <p style={{ fontSize: '0.8rem', color: 'var(--warning)', fontStyle: 'italic', margin: 0 }}>
                      ⚠️ Administrator belum mengatur nomor WhatsApp resmi sistem. Direct WA mungkin mengharuskan Anda memilih kontak secara manual.
                    </p>
                  )}
                </div>
              </div>

              <div style={{ background: 'var(--surface-soft)', padding: '1rem 1.25rem', borderRadius: '8px', borderLeft: '4px solid var(--primary)', fontSize: '0.92rem' }}>
                <p style={{ margin: '0 0 0.5rem 0', fontWeight: 600 }}>Cara Aktivasi:</p>
                <ol style={{ margin: 0, paddingLeft: '1.25rem', lineHeight: '1.6', color: 'var(--text)' }}>
                  {status?.bot_phone_number ? (
                    <li>
                      Buka chat dengan Bot WhatsApp Pekan di nomor: <strong>{status.bot_phone_number}</strong>
                    </li>
                  ) : (
                    <li>Buka aplikasi WhatsApp Anda, atau klik tombol <strong>Buka WebUI Chat Bot</strong> di atas.</li>
                  )}
                  <li>Kirim pesan perintah (kode) yang telah disalin di atas ke ruang obrolan (chat).</li>
                  <li>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', color: 'var(--primary)' }}>
                      <span className="spinner-border text-primary" style={{ width: '12px', height: '12px', border: '2px solid currentColor', borderRightColor: 'transparent', borderRadius: '50%', display: 'inline-block', animation: 'spin 0.75s linear infinite' }} />
                      Menunggu pesan aktivasi Anda... Halaman ini akan otomatis terhubung.
                    </span>
                  </li>
                </ol>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Global CSS animation for spinner */}
      <style>{`
        @keyframes spin {
          to { transform: rotate(360deg); }
        }
      `}</style>

      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </div>
  );
}

