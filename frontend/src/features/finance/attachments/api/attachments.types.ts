export type FinanceAttachmentOwnerType = "savings" | "budgets" | "reminders";

export type FinanceAttachment = {
  id: string;
  owner_type: FinanceAttachmentOwnerType;
  owner_id: string;
  original_filename: string;
  mime_type: string;
  scan_status: string;
  size_bytes: number;
  created_at: string;
};

