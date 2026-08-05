import { useEffect, useState } from "react";
import { useDebounce } from "../../../../core/hooks/useDebounce";
import { getDashboardSeries, getDashboardSummary, getDashboardTopCategories } from "../api/dashboard.api";
import { DashboardCategoryTotal, DashboardSeriesPoint, DashboardSummary } from "../api/dashboard.types";
import { useI18n } from "../../../../core/i18n/i18n";

type DashboardState = {
  summary?: DashboardSummary;
  series: DashboardSeriesPoint[];
  topCategories: DashboardCategoryTotal[];
  error?: string;
  loading: boolean;
};

export function useDashboard(range: { from?: string; to?: string }): DashboardState {
  const { t } = useI18n();
  const [state, setState] = useState<DashboardState>({
    series: [],
    topCategories: [],
    loading: true
  });

  const debouncedFrom = useDebounce(range.from, 500);
  const debouncedTo = useDebounce(range.to, 500);
  
  useEffect(() => {
    let active = true;
    const currentRange = { from: debouncedFrom, to: debouncedTo };
    setState((prev) => ({ ...prev, loading: true, error: undefined }));
    Promise.all([
      getDashboardSummary(currentRange),
      getDashboardSeries(currentRange),
      getDashboardTopCategories({ ...currentRange, limit: "10" })
    ])
      .then(([summary, series, topCategories]) => {
        if (!active) return;
        setState({
          summary,
          series,
          topCategories,
          loading: false
        });
      })
      .catch((err) => {
        if (!active) return;
        setState((prev) => ({
          ...prev,
          loading: false,
          error: err instanceof Error ? err.message : t("errors.loadDashboardFailed")
        }));
      });

    return () => {
      active = false;
    };
  }, [debouncedFrom, debouncedTo, t]);

  return state;
}

