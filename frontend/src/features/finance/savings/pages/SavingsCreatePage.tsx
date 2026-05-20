import { useNavigate, useParams } from "react-router-dom";
import { createSavings } from "../api/savings.api";
import { SavingsForm } from "../components/SavingsForm";
import { useI18n } from "../../../../core/i18n/i18n";
import { setFlashToast } from "../../../../core/toast/flashToast";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { PageHeader } from "../../../../core/components/PageHeader";

export function SavingsCreatePage(): JSX.Element {
  const { t } = useI18n();
  const { tenantCode } = useParams();
  const navigate = useNavigate();
  const { toasts, error: showError, remove } = useToast();

  async function handleSubmit(payload: Parameters<typeof createSavings>[0]): Promise<void> {
    try {
      await createSavings(payload);
      setFlashToast({ message: t("common.saveSuccess"), type: "success" });
      navigate(`/app/${tenantCode ?? "default"}/finance/savings`, { replace: true });
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.loadSavingsFailed"));
      throw err;
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("savings.form.create")} 
        description={t("savings.subtitle")}
      />

      <div className="narrow-stack">
        <SavingsForm 
          submitLabel={t("savings.form.createBtn")} 
          onSubmit={handleSubmit} 
          onCancel={() => navigate("..")}
        />
      </div>
      <ToastContainer toasts={toasts} onRemove={remove} />
    </section>
  );
}

