import { apiFetch } from "../../../../core/api/client";
import { FinanceAttachment, FinanceAttachmentOwnerType } from "./attachments.types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

export async function listFinanceAttachments(ownerType: FinanceAttachmentOwnerType, ownerID: string): Promise<FinanceAttachment[]> {
  const params = new URLSearchParams({
    owner_type: ownerType,
    owner_id: ownerID
  });
  const data = await apiFetch<{ items: FinanceAttachment[] }>(`/finance/attachments?${params.toString()}`);
  return Array.isArray(data?.items) ? data.items : [];
}

export async function uploadFinanceAttachments(
  ownerType: FinanceAttachmentOwnerType,
  ownerID: string,
  files: File[]
): Promise<FinanceAttachment[]> {
  if (files.length === 0) {
    return [];
  }
  const createPayload = (mode: "bulk" | "single", file?: File): FormData => {
    const payload = new FormData();
    payload.set("owner_type", ownerType);
    payload.set("owner_id", ownerID);
    if (mode === "single" && file) {
      // Compatibility with legacy handler variants.
      payload.append("file", file);
      payload.append("files", file);
      return payload;
    }
    files.forEach((item) => payload.append("files", item));
    return payload;
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
    const data = await apiFetch<{ items: FinanceAttachment[] }>("/finance/attachments", {
      method: "POST",
      body: createPayload("bulk")
    });
    return Array.isArray(data?.items) ? data.items : [];
  } catch (err) {
    if (!isRecoverableBulkUploadError(err)) {
      throw err;
    }

    const uploaded: FinanceAttachment[] = [];
    let firstFatal: Error | null = null;
    for (const file of files) {
      try {
        const data = await apiFetch<{ items: FinanceAttachment[] }>("/finance/attachments", {
          method: "POST",
          body: createPayload("single", file)
        });
        uploaded.push(...(data.items ?? []));
      } catch (singleErr) {
        if (!isRecoverableBulkUploadError(singleErr) && singleErr instanceof Error && !firstFatal) {
          firstFatal = singleErr;
        }
      }
    }
    if (firstFatal) {
      throw firstFatal;
    }
    return uploaded;
  }
}

export async function deleteFinanceAttachment(
  ownerType: FinanceAttachmentOwnerType,
  ownerID: string,
  attachmentID: string
): Promise<void> {
  const params = new URLSearchParams({
    owner_type: ownerType,
    owner_id: ownerID
  });
  await apiFetch(`/finance/attachments/${attachmentID}?${params.toString()}`, {
    method: "DELETE"
  });
}

export function financeAttachmentViewURL(ownerType: FinanceAttachmentOwnerType, ownerID: string, attachmentID: string): string {
  const params = new URLSearchParams({
    owner_type: ownerType,
    owner_id: ownerID,
    disposition: "inline"
  });
  return `${API_BASE_URL}/finance/attachments/${attachmentID}/download?${params.toString()}`;
}

export async function openFinanceAttachment(ownerType: FinanceAttachmentOwnerType, ownerID: string, attachmentID: string): Promise<void> {
  const response = await fetch(financeAttachmentViewURL(ownerType, ownerID, attachmentID), {
    credentials: "include"
  });
  if (!response.ok) {
    throw new Error("Failed to open attachment");
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  window.open(url, "_blank", "noopener,noreferrer");
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}
