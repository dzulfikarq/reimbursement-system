import { Outlet } from "react-router-dom";
import AppSidebar from "./AppSidebar";
import AppHeader from "./AppHeader";
import Backdrop from "./Backdrop";
import { useUIStore } from "../../stores/ui";

// Authenticated app shell (TailAdmin structure): fixed sidebar + sticky
// header + scrollable content. Session wiring arrives in M1.
export default function AppLayout() {
  const expanded = useUIStore((s) => s.sidebarExpanded);
  return (
    <div className="min-h-screen">
      <Backdrop />
      <AppSidebar />
      <div
        className={`transition-[padding] duration-300 ${
          expanded ? "lg:pl-[290px]" : "lg:pl-[90px]"
        }`}
      >
        <AppHeader />
        <main className="mx-auto w-full max-w-[1600px] p-4 md:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
