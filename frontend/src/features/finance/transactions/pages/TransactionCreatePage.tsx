import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { createTransaction, uploadTransactionAttachments } from "../api/transaction.api";
import { CreateTransactionPayload } from "../api/transaction.types";
import { TransactionForm } from "../components/TransactionForm";
import { useTransactionMasterData } from "../hooks/useTransactionMasterData";
import { useI18n } from "../../../../core/i18n/i18n";
import { getMeProfile } from "../../../../core/auth/auth-api";
import { setFlashToast } from "../../../../core/toast/flashToast";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { PageHeader } from "../../../../core/components/PageHeader";

function restoreReceiptFile(): File | null {
  try {
    const raw = window.sessionStorage.getItem("pekan_receipt_attach_file");
    if (!raw) return null;
    const parsed = JSON.parse(raw) as { name: string; type: string; dataURL: string };
    if (!parsed.dataURL) return null;
    const arr = parsed.dataURL.split(",");
    const bstr = atob(arr[1]);
    const u8arr = new Uint8Array(bstr.length);
    for (let i = 0; i < bstr.length; i++) {
      u8arr[i] = bstr.charCodeAt(i);
    }
    return new File([u8arr], parsed.name || "receipt.jpg", { type: parsed.type || "image/jpeg" });
  } catch {
    return null;
  }
}

export function TransactionCreatePage(): JSX.Element {
  const navigate = useNavigate();
  const { tenantCode } = useParams();
  const { accounts, categories, savings, loading, error } = useTransactionMasterData();
  const { t } = useI18n();
  const { toasts, error: showError, remove } = useToast();
  const [currentUserLabel, setCurrentUserLabel] = useState<string>("-");
  const receiptDraft = useMemo(() => {
    if (typeof window === "undefined") return null;
    const raw = window.sessionStorage.getItem("pekan_receipt_draft");
    if (!raw) return null;
    try {
      return JSON.parse(raw) as Partial<CreateTransactionPayload> & { category_name?: string };
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    getMeProfile()
      .then((profile) => setCurrentUserLabel(profile.full_name || profile.username || profile.email || "-"))
      .catch(() => setCurrentUserLabel("-"));
  }, []);

  const isIgnorableAttachmentError = (err: unknown): boolean => {
    const message = err instanceof Error ? err.message.toLowerCase() : "";
    return (
      /f(?:i)?le.*required|required.*f(?:i)?le/.test(message) ||
      /f\w*\s*(is|are)?\s*required/.test(message) ||
      message.includes("file is required") ||
      message.includes("file required") ||
      message.includes("file_required") ||
      message.includes("fle is required") ||
      message.includes("invalid multipart") ||
      message.includes("bad request to api")
    );
  };

  const receiptFile = useMemo(() => restoreReceiptFile(), []);

  async function handleSubmit(input: { payload: CreateTransactionPayload; files: File[] }): Promise<void> {
    try {
      const created = await createTransaction(input.payload);
      const allFiles = [...input.files];
      if (!input.payload.receipt_scan_id) {
        if (receiptFile && !allFiles.some((f) => f.name === receiptFile.name && f.size === receiptFile.size)) {
          allFiles.push(receiptFile);
        }
      }
      if (allFiles.length > 0) {
        try {
          await uploadTransactionAttachments(created.id, allFiles);
        } catch (err) {
          if (!isIgnorableAttachmentError(err)) {
            // Attachment upload should not block transaction creation.
          }
        }
      }
      if (typeof window !== "undefined") {
        window.sessionStorage.removeItem("pekan_receipt_draft");
        window.sessionStorage.removeItem("pekan_receipt_attach_file");
      }
      setFlashToast({ message: t("common.saveSuccess"), type: "success" });
      if (tenantCode) {
        navigate(`/app/${tenantCode}/finance/transactions`, { replace: true });
        return;
      }
      navigate("..");
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.saveTransactionFailed"));
      throw err;
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("transactions.create")} 
        description={t("transactions.subtitle")}
      />
      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}
      {!loading && !error ? (
        <div className="narrow-stack">
          <TransactionForm
            onSubmit={handleSubmit}
            onCancel={() => navigate("..")}
            accounts={accounts}
            categories={categories}
            savingsOptions={savings}
            actorLabel={currentUserLabel}
            initialValue={receiptDraft ?? undefined}
            initialCategoryName={receiptDraft?.category_name}
            initialFiles={receiptFile ? [receiptFile] : []}
          />
        </div>
      ) : null}
      <ToastContainer toasts={toasts} onRemove={remove} />
    </section>
  );
}

