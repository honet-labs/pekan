import { apiFetch } from "../../../../core/api/client";
import { DashboardCategoryTotal, DashboardSeriesPoint, DashboardSummary } from "./dashboard.types";

function buildQuery(params: Record<string, string | undefined>): string {
  const entries = Object.entries(params).filter(([, value]) => value);
  if (!entries.length) {
    return "";
  }
  const search = new URLSearchParams();
  entries.forEach(([key, value]) => {
    if (value) {
      search.set(key, value);
    }
  });
  return `?${search.toString()}`;
}

export async function getDashboardSummary(params: { from?: string; to?: string }): Promise<DashboardSummary> {
  return apiFetch<DashboardSummary>(`/finance/dashboard/summary${buildQuery(params)}`);
}

export async function getDashboardSeries(params: { from?: string; to?: string }): Promise<DashboardSeriesPoint[]> {
  const data = await apiFetch<{ items: DashboardSeriesPoint[] }>(`/finance/dashboard/series${buildQuery(params)}`);
  return Array.isArray(data?.items) ? data.items : [];
}

export async function getDashboardTopCategories(params: { from?: string; to?: string; limit?: string }): Promise<DashboardCategoryTotal[]> {
  const data = await apiFetch<{ items: DashboardCategoryTotal[] }>(`/finance/dashboard/top-categories${buildQuery(params)}`);
  return Array.isArray(data?.items) ? data.items : [];
}
