import { apiFetch } from "../../../../core/api/client";
import { CreateTransactionsReportPayload, Report } from "./reports.types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

type ListReportsParams = {
  page?: number;
  page_size?: number;
};

type ListReportsResponse = {
  items: Report[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
};

export async function listReports(params: ListReportsParams = {}): Promise<ListReportsResponse> {
  const query = new URLSearchParams();
  if (params.page) query.set("page", String(params.page));
  if (params.page_size) query.set("page_size", String(params.page_size));

  const queryString = query.toString() ? `?${query.toString()}` : "";
  const data = await apiFetch<ListReportsResponse>(`/finance/reports${queryString}`);
  
  return {
    ...data,
    items: Array.isArray(data?.items) ? data.items : []
  };
}

export async function createTransactionsReport(payload: CreateTransactionsReportPayload): Promise<Report> {
  return apiFetch<Report>("/finance/reports/transactions", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export async function downloadReport(report: Report): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/finance/reports/${report.id}/download`, {
    credentials: "include"
  });
  if (!response.ok) {
    throw new Error("Failed to download report");
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `${report.report_type}.${report.format}`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

export async function deleteReport(reportID: string): Promise<void> {
  await apiFetch(`/finance/reports/${reportID}`, {
    method: "DELETE"
  });
}
