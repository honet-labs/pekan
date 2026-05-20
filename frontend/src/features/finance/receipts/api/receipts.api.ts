import { apiFetch, API_BASE_URL } from "../../../../core/api/client";
import { ReceiptConfigStatus, ReceiptProviderConfig, ReceiptProviderTestResult, ReceiptScanHistoryItem, ReceiptScanResult } from "./receipts.types";

export async function listReceiptProviders(): Promise<ReceiptProviderConfig[]> {
  const data = await apiFetch<{ items: ReceiptProviderConfig[] }>("/finance/settings/receipt-scan/providers");
  return Array.isArray(data?.items) ? data.items : [];
}

export async function updateReceiptProviders(items: Array<{ provider_code: string; display_name: string; base_url?: string; model_name: string; is_enabled: boolean; api_key?: string; clear_api_key?: boolean }>): Promise<ReceiptProviderConfig[]> {
  const data = await apiFetch<{ items: ReceiptProviderConfig[] }>("/finance/settings/receipt-scan/providers", {
    method: "PUT",
    body: JSON.stringify({ items })
  });
  return Array.isArray(data?.items) ? data.items : [];
}

export async function testReceiptProviderConnection(input: { provider_code: string; base_url?: string; api_key?: string; model_name?: string }): Promise<ReceiptProviderTestResult> {
  return apiFetch<ReceiptProviderTestResult>("/finance/settings/receipt-scan/providers/test", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function getReceiptConfigStatus(): Promise<ReceiptConfigStatus> {
  return apiFetch<ReceiptConfigStatus>("/finance/receipt-scan/status");
}

export async function listReceiptScanHistory(limit = 10): Promise<ReceiptScanHistoryItem[]> {
  const data = await apiFetch<{ items: ReceiptScanHistoryItem[] }>(`/finance/receipt-scan/history?limit=${limit}`);
  return Array.isArray(data?.items) ? data.items : [];
}

export async function scanReceipt(providerCode: string, file: File): Promise<ReceiptScanResult> {
  const form = new FormData();
  form.append("provider_code", providerCode);
  form.append("file", file);
  return apiFetch<ReceiptScanResult>("/finance/receipt-scan/scan", {
    method: "POST",
    body: form
  });
}

export async function deleteReceiptScanHistoryItem(scanID: string): Promise<void> {
  await apiFetch(`/finance/receipt-scan/history/${scanID}`, { method: "DELETE" });
}

export async function clearReceiptScanHistory(): Promise<void> {
  await apiFetch("/finance/receipt-scan/history", { method: "DELETE" });
}

export function receiptScanImageURL(scanID: string): string {
  return `${API_BASE_URL}/finance/receipt-scan/history/${scanID}/image`;
}

export async function fetchReceiptScanImageBlob(scanID: string): Promise<string> {
  const response = await fetch(receiptScanImageURL(scanID), {
    credentials: "include"
  });
  if (!response.ok) {
    throw new Error("Failed to load receipt image");
  }
  const blob = await response.blob();
  return URL.createObjectURL(blob);
}
