import { apiFetch, API_BASE_URL } from "../../../../core/api/client";
import {
  CreateTransactionPayload,
  Transaction,
  TransactionAttachment,
  TransactionType,
  UpdateTransactionPayload
} from "./transaction.types";

type ListResponse = {
  items: Transaction[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
};

type ListTransactionsParams = {
  page?: number;
  page_size?: number;
  type?: TransactionType;
  from?: string;
  to?: string;
  q?: string;
  category_id?: string;
};

type TransactionWire = Omit<Transaction, "tid" | "input_date"> & {
  tid?: string;
  input_date?: string;
};

function toShortID(id: string): string {
  if (!id) {
    return "";
  }
  if (id.length < 8) {
    return id.toUpperCase();
  }
  return id.slice(0, 8).toUpperCase();
}

function toDateOnly(value?: string): string {
  if (!value) {
    return new Date().toISOString().slice(0, 10);
  }
  return value.slice(0, 10);
}

function normalizeTransaction(raw: TransactionWire): Transaction {
  return {
    ...raw,
    tid: raw.tid ?? toShortID(raw.id),
    input_date: toDateOnly(raw.input_date ?? raw.created_at),
    transaction_date: toDateOnly(raw.transaction_date),
    savings_ids: Array.isArray(raw.savings_ids) ? raw.savings_ids : [],
    savings_names: Array.isArray(raw.savings_names) ? raw.savings_names : [],
    items: Array.isArray((raw as TransactionWire & { items?: unknown[] }).items)
      ? ((raw as TransactionWire & { items?: unknown[] }).items as unknown[]).map((item) => ({
          id: typeof (item as { id?: unknown }).id === "string" ? (item as { id?: string }).id : undefined,
          item_name: String((item as { item_name?: unknown }).item_name ?? ""),
          quantity: Number((item as { quantity?: unknown }).quantity ?? 0),
          price_per_unit_minor: Number((item as { price_per_unit_minor?: unknown }).price_per_unit_minor ?? 0),
          discount_minor: Number((item as { discount_minor?: unknown }).discount_minor ?? 0),
          total_minor: Number((item as { total_minor?: unknown }).total_minor ?? 0),
          notes: typeof (item as { notes?: unknown }).notes === "string" ? (item as { notes?: string }).notes : undefined
        }))
      : []
  };
}

function buildQuery(params: ListTransactionsParams): string {
  const query = new URLSearchParams();
  if (params.page) query.set("page", String(params.page));
  if (params.page_size) query.set("page_size", String(params.page_size));
  if (params.type) query.set("type", params.type);
  if (params.from) query.set("from", params.from);
  if (params.to) query.set("to", params.to);
  if (params.q) query.set("q", params.q);
  if (params.category_id) query.set("category_id", params.category_id);
  const text = query.toString();
  return text ? `?${text}` : "";
}

export function listTransactions(params: ListTransactionsParams = {}): Promise<ListResponse> {
  return apiFetch<ListResponse>(`/finance/transactions${buildQuery(params)}`).then((result) => ({
    ...result,
    items: result.items.map((item) => normalizeTransaction(item as TransactionWire))
  }));
}

export function listTransactionsBySavings(savingsID: string): Promise<Transaction[]> {
  return apiFetch<{ items: TransactionWire[] }>(`/finance/transactions/by-savings/${savingsID}`).then((result) =>
    (result.items || []).map((item) => normalizeTransaction(item))
  );
}

export function getTransaction(transactionID: string): Promise<Transaction> {
  return apiFetch<TransactionWire>(`/finance/transactions/${transactionID}`).then(normalizeTransaction);
}

export function createTransaction(payload: CreateTransactionPayload): Promise<Transaction> {
  return apiFetch<TransactionWire>("/finance/transactions", {
    method: "POST",
    body: JSON.stringify(payload)
  }).then(normalizeTransaction);
}

export function updateTransaction(transactionID: string, payload: UpdateTransactionPayload): Promise<Transaction> {
  return apiFetch<TransactionWire>(`/finance/transactions/${transactionID}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  }).then(normalizeTransaction);
}

export async function deleteTransaction(transactionID: string): Promise<void> {
  await apiFetch(`/finance/transactions/${transactionID}`, { method: "DELETE" });
}

export async function uploadTransactionAttachments(transactionID: string, files: File[]): Promise<void> {
  if (files.length === 0) {
    return;
  }
  const createPayload = (mode: "bulk" | "single", file?: File): FormData => {
    const body = new FormData();
    if (mode === "single" && file) {
      // Compatibility: some server revisions only accept `file`,
      // while newer handlers support `files` (multi-part list).
      body.append("file", file);
      body.append("files", file);
      return body;
    }
    files.forEach((item) => body.append("files", item));
    return body;
  };

  const isRecoverableBulkUploadError = (err: unknown): boolean => {
    const message = err instanceof Error ? err.message.toLowerCase() : "";
    const hasFileRequiredPattern =
      /f(?:i)?le.*required|required.*f(?:i)?le/.test(message) ||
      /f\w*\s*(is|are)?\s*required/.test(message);
    return (
      hasFileRequiredPattern ||
      message.includes("file required") ||
      message.includes("file_required") ||
      message.includes("file is required") ||
      message.includes("fle is required") ||
      message.includes("at least one file is required") ||
      message.includes("invalid multipart") ||
      message.includes("bad request to api") ||
      message.includes("invalid request")
    );
  };

  try {
    await apiFetch(`/finance/transactions/${transactionID}/attachments`, {
      method: "POST",
      body: createPayload("bulk")
    });
  } catch (err) {
    if (!isRecoverableBulkUploadError(err)) {
      throw err;
    }
    // Compatibility fallback for servers that only accept one file part/key per request.
    let firstFatal: Error | null = null;
    for (const file of files) {
      try {
        await apiFetch(`/finance/transactions/${transactionID}/attachments`, {
          method: "POST",
          body: createPayload("single", file)
        });
      } catch (singleErr) {
        if (!isRecoverableBulkUploadError(singleErr) && singleErr instanceof Error && !firstFatal) {
          firstFatal = singleErr;
        }
      }
    }
    if (firstFatal) {
      throw firstFatal;
    }
  }
}

export async function listTransactionAttachments(transactionID: string): Promise<TransactionAttachment[]> {
  const data = await apiFetch<{ items: TransactionAttachment[] }>(`/finance/transactions/${transactionID}/attachments`);
  return Array.isArray(data?.items) ? data.items : [];
}

export function transactionAttachmentViewURL(transactionID: string, attachmentID: string): string {
  return `${API_BASE_URL}/finance/transactions/${transactionID}/attachments/${attachmentID}/download?view=1`;
}

export async function openTransactionAttachment(transactionID: string, attachmentID: string): Promise<void> {
  const url = transactionAttachmentViewURL(transactionID, attachmentID);
  const newWindow = window.open(url, "_blank", "noopener,noreferrer");
  if (!newWindow) {
    throw new Error("Popup blocked");
  }
}

