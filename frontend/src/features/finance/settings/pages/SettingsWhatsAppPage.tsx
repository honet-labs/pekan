import { WhatsAppIntegration } from "../components/WhatsAppIntegration";
import { PageHeader } from "../../../../core/components/PageHeader";
import { useI18n } from "../../../../core/i18n/i18n";

export function SettingsWhatsAppPage(): JSX.Element {
  const { t } = useI18n();

  return (
    <section className="page-section">
      <PageHeader 
        title={t("settings.nav.whatsapp")} 
        description="Kelola integrasi WhatsApp Bot AI Anda secara mandiri dengan metode OTP atau koneksi langsung." 
      />
      <div style={{ marginTop: "1.5rem" }}>
        <WhatsAppIntegration />
      </div>
    </section>
  );
}
