import { NavLink } from "react-router-dom";
import {
  LayoutDashboard,
  ReceiptText,
  ClipboardCheck,
  Banknote,
  Users,
  Tags,
  type LucideIcon,
} from "lucide-react";
import { useUIStore } from "../../stores/ui";
import { useAuthStore } from "../../stores/auth";

interface NavItem {
  name: string;
  icon: LucideIcon;
  path: string;
  roles: string[];
}

interface NavSection {
  label: string;
  items: NavItem[];
}

// Role-scoped navigation (docs/05 sidebar matrix). Backend still authorizes.
const NAV_SECTIONS: NavSection[] = [
  {
    label: "Menu",
    items: [
      { name: "Dashboard", icon: LayoutDashboard, path: "/", roles: ["employee", "manager", "finance", "admin"] },
      { name: "My Claims", icon: ReceiptText, path: "/reimbursements", roles: ["employee", "manager", "finance", "admin"] },
      { name: "Approvals", icon: ClipboardCheck, path: "/approvals", roles: ["manager", "finance", "admin"] },
      { name: "Payments", icon: Banknote, path: "/payments", roles: ["finance"] },
    ],
  },
  {
    label: "Master Data",
    items: [
      { name: "Categories", icon: Tags, path: "/admin/categories", roles: ["admin"] },
    ],
  },
  {
    label: "Administration",
    items: [{ name: "Users", icon: Users, path: "/admin/users", roles: ["admin"] }],
  },
];

export default function AppSidebar() {
  const sidebarExpanded = useUIStore((s) => s.sidebarExpanded);
  const sidebarMobileOpen = useUIStore((s) => s.sidebarMobileOpen);
  const role = useAuthStore((s) => s.user?.role);

  const sections = NAV_SECTIONS.map((s) => ({
    ...s,
    // Default-deny: no role yet → nothing renders.
    items: s.items.filter((item) => !!role && item.roles.includes(role)),
  })).filter((s) => s.items.length > 0);

  return (
    <aside
      className={`fixed left-0 top-0 z-50 flex h-screen flex-col border-r border-gray-200 bg-white px-5 transition-all duration-300 ease-in-out dark:border-gray-800 dark:bg-gray-900 ${
        sidebarExpanded ? "lg:w-[290px]" : "lg:w-[90px]"
      } ${sidebarMobileOpen ? "w-[290px] translate-x-0" : "-translate-x-full"} lg:translate-x-0`}
    >
      <div
        className={`flex py-8 ${sidebarExpanded ? "justify-start" : "lg:justify-center"} justify-start`}
      >
        <NavLink to="/" className="flex items-center gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-brand-500 text-white shadow-theme-xs">
            <ReceiptText className="size-5" />
          </div>
          {(sidebarExpanded || sidebarMobileOpen) && (
            <span className="text-lg font-semibold text-gray-800 dark:text-white/90">
              Reimburse<span className="text-brand-500">Flow</span>
            </span>
          )}
        </NavLink>
      </div>

      <nav className="no-scrollbar flex-1 overflow-y-auto">
        {sections.map((section, si) => (
          <ul key={section.label} className={`mb-6 flex flex-col gap-1.5 ${si > 0 ? "pt-4 border-t border-gray-100 dark:border-gray-800" : ""}`}>
            <li
              className={`menu-item px-0 pb-2 text-theme-xs font-semibold uppercase tracking-wider text-gray-400 ${
                sidebarExpanded || sidebarMobileOpen ? "" : "lg:sr-only"
              }`}
            >
              {section.label}
            </li>
            {section.items.map(({ name, icon: Icon, path }) => (
              <li key={name}>
                <NavLink
                  to={path}
                  end={path === "/"}
                  onClick={() => useUIStore.getState().setSidebarMobileOpen(false)}
                  className={({ isActive }) =>
                    `menu-item group ${isActive ? "menu-item-active" : "menu-item-inactive"} ${
                      sidebarExpanded ? "" : "lg:justify-center"
                    }`
                  }
                >
                  {({ isActive }) => (
                    <>
                      <Icon
                        className={`size-5 shrink-0 ${
                          isActive ? "menu-item-icon-active" : "menu-item-icon-inactive"
                        }`}
                      />
                      {(sidebarExpanded || sidebarMobileOpen) && name}
                    </>
                  )}
                </NavLink>
              </li>
            ))}
          </ul>
        ))}
      </nav>
    </aside>
  );
}
