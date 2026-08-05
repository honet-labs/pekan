import { useEffect, useMemo, useState } from "react";
import { useI18n } from "../../../../core/i18n/i18n";
import { listRelatedSavingsTransactions, SavingsRelatedTransaction } from "../api/savings.api";

interface RelatedTransactionsPanelProps {
  savingsID: string;
}

function formatTransactionDate(value: string, locale: string): string {
  if (!value) {
    return "-";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  return parsed.toLocaleDateString(locale === "id" ? "id-ID" : "en-US");
}

export function RelatedTransactionsPanel({ savingsID }: RelatedTransactionsPanelProps): JSX.Element {
  const { t, locale } = useI18n();
  const [transactions, setTransactions] = useState<SavingsRelatedTransaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US"),
    [locale]
  );

  useEffect(() => {
    const loadRelatedTransactions = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await listRelatedSavingsTransactions(savingsID);
        setTransactions(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : t("errors.loadSavingsFailed"));
        setTransactions([]);
      } finally {
        setLoading(false);
      }
    };

    if (savingsID) {
      loadRelatedTransactions().catch(() => undefined);
    }
  }, [savingsID, t]);

  if (loading) {
    return <p className="page-subtitle">{t("common.loading")}</p>;
  }

  return (
    <div className="card surface">
      <h3 className="form-title">{t("savings.relatedTransactions")}</h3>
      {error ? (
        <p className="alert error">{error}</p>
      ) : transactions.length > 0 ? (
        <div className="data-table-wrap table-mobile-stack">
          <table className="data-table">
            <thead>
              <tr>
                <th>TID</th>
                <th>{t("transactions.table.date")}</th>
                <th>{t("transactions.table.category")}</th>
                <th>{t("transactions.table.total")}</th>
                <th>{t("transactions.table.description")}</th>
              </tr>
            </thead>
            <tbody>
              {transactions.map((tx) => (
                <tr key={tx.id}>
                  <td data-label="TID">{tx.tid || tx.id || "-"}</td>
                  <td data-label={t("transactions.table.date")}>{formatTransactionDate(tx.transaction_date, locale)}</td>
                  <td data-label={t("transactions.table.category")}>{tx.category_name || t("dashboard.uncategorized")}</td>
                  <td data-label={t("transactions.table.total")}>{`${tx.currency || "IDR"} ${numberFormatter.format(Number(tx.amount_minor || 0))}`}</td>
                  <td data-label={t("transactions.table.description")}>{tx.description || "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="page-subtitle">{t("common.noData")}</p>
      )}
    </div>
  );
}
