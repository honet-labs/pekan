import { useNavigate, useParams } from "react-router-dom";
import { createBudget } from "../api/budgets.api";
import { useFinanceMasterData } from "../../masterdata/hooks/useFinanceMasterData";
import { BudgetForm } from "../components/BudgetForm";
import { useI18n } from "../../../../core/i18n/i18n";
import { setFlashToast } from "../../../../core/toast/flashToast";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { PageHeader } from "../../../../core/components/PageHeader";

export function BudgetCreatePage(): JSX.Element {
  const { t } = useI18n();
  const { tenantCode } = useParams();
  const navigate = useNavigate();
  const { categories, loading, error } = useFinanceMasterData();
  const { toasts, error: showError, remove } = useToast();

  async function handleSubmit(payload: Parameters<typeof createBudget>[0]): Promise<void> {
    try {
      await createBudget(payload);
      setFlashToast({ message: t("common.saveSuccess"), type: "success" });
      navigate(`/app/${tenantCode ?? "default"}/finance/budgets`, { replace: true });
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.loadBudgetsFailed"));
      throw err;
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("budgets.form.create")} 
        description={t("budgets.subtitle")}
      />

      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      {!loading ? (
        <div className="narrow-stack">
          <BudgetForm 
            categories={categories} 
            submitLabel={t("budgets.form.createBtn")} 
            onSubmit={handleSubmit} 
            onCancel={() => navigate("..")}
          />
        </div>
      ) : null}
      <ToastContainer toasts={toasts} onRemove={remove} />
    </section>
  );
}

