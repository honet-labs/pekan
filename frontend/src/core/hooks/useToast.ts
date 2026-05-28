import { useState, useCallback } from "react";
import { ToastType } from "../components/Toast";

export interface Toast {
  id: string;
  message: string;
  type: ToastType;
  duration?: number;
}

export function useToast() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const show = useCallback((message: string, type: ToastType = "info", duration: number = 4000) => {
    const id = `toast-${Date.now()}-${Math.random()}`;
    const toast: Toast = { id, message, type, duration };
    setToasts((prev) => [...prev, toast]);
    return id;
  }, []);

  const success = useCallback((message: string) => {
    return show(message, "success", 3000);
  }, [show]);

  const error = useCallback((message: string) => {
    console.error(`[App Error] ${message}`);
    return show(message, "error", 5000);
  }, [show]);

  const info = useCallback((message: string) => {
    return show(message, "info", 4000);
  }, [show]);

  const warning = useCallback((message: string) => {
    console.warn(`[App Warning] ${message}`);
    return show(message, "warning", 4000);
  }, [show]);

  const remove = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const removeAll = useCallback(() => {
    setToasts([]);
  }, []);

  return {
    toasts,
    show,
    success,
    error,
    info,
    warning,
    remove,
    removeAll,
  };
}
