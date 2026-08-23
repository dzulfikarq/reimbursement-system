import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";

// Cookie-based auth: withCredentials sends access/refresh cookies; tokens
// never touch JS storage (docs/06). CSRF cookie is the only JS-readable one.
export const api = axios.create({
  baseURL: "/api/v1",
  withCredentials: true,
});

export function getCookie(name: string): string | undefined {
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : undefined;
}

api.interceptors.request.use((config) => {
  if (["post", "put", "patch", "delete"].includes(config.method ?? "")) {
    const token = getCookie("csrf_token");
    if (token) config.headers["X-CSRF-Token"] = token;
  }
  return config;
});

// Single-flight refresh (docs/05): concurrent 401s share one refresh call;
// replayed exactly once — no infinite retry.
let refreshPromise: Promise<unknown> | null = null;

api.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const original = error.config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined;
    const status = error.response?.status;
    const isRefreshCall = original?.url?.endsWith("/auth/refresh");

    if (status === 401 && original && !original._retried && !isRefreshCall) {
      if (!refreshPromise) {
        refreshPromise = api.post("/auth/refresh").finally(() => {
          refreshPromise = null;
        });
      }
      try {
        await refreshPromise;
        original._retried = true;
        return api(original);
      } catch {
        // fall through to rejection handling below
      }
    }

    return Promise.reject(error);
  },
);

// Fixed envelope shapes (docs/04).
export interface ApiEnvelope<T> {
  success: boolean;
  data?: T;
  message?: string;
}

export interface ApiErrorBody {
  code: string;
  message: string;
  details?: { field: string; message: string }[];
}