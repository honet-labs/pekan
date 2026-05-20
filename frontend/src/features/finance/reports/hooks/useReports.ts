import { useEffect, useState } from "react";
import { createTransactionsReport, deleteReport, listReports } from "../api/reports.api";
import { CreateTransactionsReportPayload, Report } from "../api/reports.types";
import { useI18n } from "../../../../core/i18n/i18n";

export function useReports() {
  const { t } = useI18n();
  const [items, setItems] = useState<Report[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [total, setTotal] = useState(0);

  const refresh = () => {
    setLoading(true);
    listReports({ page, page_size: pageSize })
      .then((res) => {
        setItems(res.items);
        setTotal(res.pagination.total);
      })
      .catch((err) => setError(err instanceof Error ? err.message : t("errors.loadReportsFailed")))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    refresh();
  }, [t, page, pageSize]);

  const create = async (payload: CreateTransactionsReportPayload) => {
    const created = await createTransactionsReport(payload);
    refresh();
    return created;
  };

  const remove = async (reportID: string) => {
    await deleteReport(reportID);
    refresh();
  };

  return { items, loading, error, page, pageSize, total, setPage, create, refresh, remove };
}

