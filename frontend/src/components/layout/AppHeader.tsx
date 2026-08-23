import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Menu, Moon, Sun, LogOut, UserCircle2 } from "lucide-react";
import { useUIStore } from "../../stores/ui";

interface AppHeaderProps {
  user?: { name: string; role: string };
  onLogout?: () => void;
}

// Simplified TailAdmin header: sidebar toggles + theme toggle + user menu.
export default function AppHeader({ user, onLogout }: AppHeaderProps) {
  const navigate = useNavigate();
  const theme = useUIStore((s) => s.theme);
  const toggleTheme = useUIStore((s) => s.toggleTheme);
  const toggleSidebar = useUIStore((s) => s.toggleSidebar);
  const setSidebarMobileOpen = useUIStore((s) => s.setSidebarMobileOpen);
  const sidebarMobileOpen = useUIStore((s) => s.sidebarMobileOpen);
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  const handleToggle = () => {
    if (window.innerWidth >= 1024) {
      toggleSidebar();
    } else {
      setSidebarMobileOpen(!sidebarMobileOpen);
    }
  };

  return (
    <header className="sticky top-0 z-40 flex w-full border-b border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
      <div className="flex w-full items-center justify-between gap-2 px-3 py-3 sm:px-6 lg:py-4">
        <button
          onClick={handleToggle}
          aria-label="Toggle Sidebar"
          className="flex size-10 items-center justify-center rounded-lg text-gray-500 dark:text-gray-400 lg:size-11 lg:border lg:border-gray-200 dark:lg:border-gray-800"
        >
          <Menu className="size-6" />
        </button>

        <div className="flex items-center gap-3">
          <button
            onClick={toggleTheme}
            aria-label="Toggle Theme"
            className="flex size-10 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-white/[0.05]"
          >
            {theme === "dark" ? <Sun className="size-5" /> : <Moon className="size-5" />}
          </button>

          <div className="relative">
            <button
              onClick={() => setUserMenuOpen((v) => !v)}
              className="flex items-center gap-2 rounded-lg p-1.5 text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-white/[0.05]"
            >
              <UserCircle2 className="size-8 text-gray-400" />
              {user && (
                <span className="hidden text-left sm:block">
                  <span className="block text-sm font-medium">{user.name}</span>
                  <span className="block text-xs capitalize text-gray-400">
                    {user.role}
                  </span>
                </span>
              )}
            </button>

            {userMenuOpen && (
              <>
                <div
                  className="fixed inset-0 z-10"
                  onClick={() => setUserMenuOpen(false)}
                />
                <div className="absolute right-0 z-20 mt-2 w-48 rounded-xl border border-gray-200 bg-white p-2 shadow-theme-lg dark:border-gray-800 dark:bg-gray-900">
                  <button
                    onClick={() => {
                      setUserMenuOpen(false);
                      onLogout?.() ?? navigate("/login");
                    }}
                    className="menu-item menu-item-inactive"
                  >
                    <LogOut className="size-4" /> Sign Out
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}
