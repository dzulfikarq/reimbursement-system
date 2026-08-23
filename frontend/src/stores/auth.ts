import { create } from "zustand";
import { api, type ApiEnvelope } from "../lib/api";

export interface User {
  id: string;
  name: string;
  email: string;
  role: "employee" | "manager" | "finance" | "admin";
  department_id?: string;
  department_name?: string;
}

interface AuthState {
  user: User | null;
  status: "idle" | "loading" | "ready";
  fetchSession: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

// Session lives in HttpOnly cookies — this store only mirrors who we are.
export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  status: "idle",

  fetchSession: async () => {
    set({ status: "loading" });
    try {
      const res = await api.get<ApiEnvelope<{ user: User }>>("/auth/me");
      set({ user: res.data.data?.user ?? null, status: "ready" });
    } catch {
      set({ user: null, status: "ready" });
    }
  },

  login: async (email, password) => {
    const res = await api.post<ApiEnvelope<{ user: User }>>("/auth/login", {
      email,
      password,
    });
    set({ user: res.data.data?.user ?? null, status: "ready" });
  },

  logout: async () => {
    try {
      await api.post("/auth/logout");
    } finally {
      set({ user: null });
    }
  },
}));
