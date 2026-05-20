import { ToastType } from "../components/Toast";

const FLASH_TOAST_KEY = "pekan.flash.toast";

export type FlashToastPayload = {
  message: string;
  type: ToastType;
};

export function setFlashToast(payload: FlashToastPayload): void {
  if (typeof window === "undefined") {
    return;
  }
  window.sessionStorage.setItem(FLASH_TOAST_KEY, JSON.stringify(payload));
}

export function consumeFlashToast(): FlashToastPayload | null {
  if (typeof window === "undefined") {
    return null;
  }
  const raw = window.sessionStorage.getItem(FLASH_TOAST_KEY);
  if (!raw) {
    return null;
  }
  window.sessionStorage.removeItem(FLASH_TOAST_KEY);
  try {
    const parsed = JSON.parse(raw) as Partial<FlashToastPayload>;
    if (!parsed || typeof parsed.message !== "string") {
      return null;
    }
    return {
      message: parsed.message,
      type: parsed.type === "success" || parsed.type === "error" || parsed.type === "info" || parsed.type === "warning" ? parsed.type : "info"
    };
  } catch {
    return null;
  }
}
