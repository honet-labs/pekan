import { useEffect, useState } from "react";
import { getMeProfile, updateMeProfile } from "../../../../core/auth/auth-api";
import { useI18n } from "../../../../core/i18n/i18n";

type ProfileForm = {
  full_name: string;
  username: string;
  email: string;
  phone: string;
  address: string;
};

const initialForm: ProfileForm = {
  full_name: "",
  username: "",
  email: "",
  phone: "",
  address: ""
};

export function ProfilePage(): JSX.Element {
  const { locale, t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<string>("");
  const [form, setForm] = useState<ProfileForm>(initialForm);

  async function loadProfile(): Promise<void> {
    setLoading(true);
    setError(null);
    try {
      const profile = await getMeProfile();
      setForm({
        full_name: profile.full_name ?? "",
        username: profile.username ?? "",
        email: profile.email ?? "",
        phone: profile.phone ?? "",
        address: profile.address ?? ""
      });
      setLastUpdated(profile.updated_at);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadProfile().catch(() => undefined);
  }, [t]);

  async function handleSubmit(event: React.FormEvent): Promise<void> {
    event.preventDefault();
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      const updated = await updateMeProfile({
        full_name: form.full_name,
        username: form.username,
        email: form.email,
        phone: form.phone || null,
        address: form.address || null
      });
      setForm({
        full_name: updated.full_name ?? "",
        username: updated.username ?? "",
        email: updated.email ?? "",
        phone: updated.phone ?? "",
        address: updated.address ?? ""
      });
      setLastUpdated(updated.updated_at);
      setSuccess(t("profile.saved"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="page-section">
      <header className="page-header">
        <div>
          <h1 className="page-title">{t("profile.title")}</h1>
          <div className="tagline-info" title={t("profile.subtitle")}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="16" x2="12" y2="12" />
              <line x1="12" y1="8" x2="12.01" y2="8" />
            </svg>
          </div>
        </div>
      </header>

      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}
      {success ? <p className="alert info">{success}</p> : null}

      {!loading ? (
        <form className="card surface form-grid compact" onSubmit={handleSubmit}>
          <label className="form-field">
            {t("profile.fullName")}
            <input
              className="input-control"
              value={form.full_name}
              onChange={(event) => setForm({ ...form, full_name: event.target.value })}
              required
            />
          </label>
          <label className="form-field">
            {t("profile.username")}
            <input
              className="input-control"
              value={form.username}
              onChange={(event) => setForm({ ...form, username: event.target.value })}
              required
            />
          </label>
          <label className="form-field">
            {t("profile.email")}
            <input
              className="input-control"
              type="email"
              value={form.email}
              onChange={(event) => setForm({ ...form, email: event.target.value })}
              required
            />
          </label>
          <label className="form-field">
            {t("profile.phone")}
            <input
              className="input-control"
              value={form.phone}
              onChange={(event) => setForm({ ...form, phone: event.target.value })}
            />
          </label>
          <label className="form-field">
            {t("profile.address")}
            <textarea
              className="input-control textarea-control"
              value={form.address}
              onChange={(event) => setForm({ ...form, address: event.target.value })}
            />
          </label>
          {lastUpdated ? (
            <small className="page-subtitle">
              {t("profile.lastUpdated")}: {new Date(lastUpdated).toLocaleString(locale === "id" ? "id-ID" : "en-US")}
            </small>
          ) : null}
          <button className="btn btn-primary" type="submit" disabled={saving}>
            {saving ? t("common.loading") : t("profile.save")}
          </button>
        </form>
      ) : null}
    </section>
  );
}
