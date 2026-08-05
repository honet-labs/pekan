export type Notification = {
  id: string;
  notification_type: string;
  title: string;
  message: string;
  status: string;
  created_at: string;
  read_at?: string | null;
};

export type NotificationPayload = {
  notification_type: string;
  title: string;
  message: string;
  metadata?: Record<string, unknown>;
};

