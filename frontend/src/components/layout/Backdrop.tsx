import { useUIStore } from "../../stores/ui";

// Ported from TailAdmin Backdrop — closes the mobile sidebar.
export default function Backdrop() {
  const sidebarMobileOpen = useUIStore((s) => s.sidebarMobileOpen);
  const setSidebarMobileOpen = useUIStore((s) => s.setSidebarMobileOpen);

  if (!sidebarMobileOpen) return null;
  return (
    <div
      className="fixed inset-0 z-40 bg-gray-900/50 backdrop-blur-sm lg:hidden"
      onClick={() => setSidebarMobileOpen(false)}
    />
  );
}
