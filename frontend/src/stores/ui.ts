import { create } from "zustand";

type Theme = "light" | "dark";

const savedTheme =
  (localStorage.getItem("theme") as Theme | null) ??
  (window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");

function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle("dark", theme === "dark");
  localStorage.setItem("theme", theme);
}
applyTheme(savedTheme);

interface UIState {
  theme: Theme;
  toggleTheme: () => void;
  sidebarExpanded: boolean;
  sidebarMobileOpen: boolean;
  toggleSidebar: () => void;
  setSidebarMobileOpen: (open: boolean) => void;
}

export const useUIStore = create<UIState>((set, get) => ({
  theme: savedTheme,
  toggleTheme: () => {
    const next = get().theme === "dark" ? "light" : "dark";
    applyTheme(next);
    set({ theme: next });
  },
  sidebarExpanded: true,
  sidebarMobileOpen: false,
  toggleSidebar: () =>
    set((s) => ({ sidebarExpanded: !s.sidebarExpanded })),
  setSidebarMobileOpen: (open) => set({ sidebarMobileOpen: open }),
}));
