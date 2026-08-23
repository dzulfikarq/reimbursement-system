import { create } from "zustand";
import { AxiosError } from "axios";
import type { ApiErrorBody } from "../lib/api";

export type ToastType = "success" | "error" | "info";

export interface ToastItem {
  id: number;
  type: ToastType;
  message: string;
}

interface ToastState {
  toasts: ToastItem[];
  push: (type: ToastType, message: string) => void;
  dismiss: (id: number) => void;
}

let nextId = 1;

export const useToastStore = create<ToastState>((set) => ({
  toasts: [],
  push: (type, message) => {
    const id = nextId++;
    set((s) => ({ toasts: [...s.toasts.slice(-4), { id, type, message }] }));
    setTimeout(() => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })), 4500);
  },
  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}));

export const toast = {
  success: (m: string) => useToastStore.getState().push("success", m),
  info: (m: string) => useToastStore.getState().push("info", m),
  error: (m: string) => useToastStore.getState().push("error", m),
};

// Extract the backend envelope message; fall back to axios text.
export function errMessage(err: unknown): string {
  if (err instanceof AxiosError) {
    const body = err.response?.data as { error?: ApiErrorBody } | undefined;
    if (body?.error?.message) return body.error.message;
    if (err.message) return err.message;
  }
  return "Something went wrong";
}
