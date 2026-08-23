import type { ReactNode } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { useAuthStore } from "../../stores/auth";

// UX-only role guard (backend re-authorizes). Denied → /403.
export default function RoleRoute({ allow, children }: { allow: string[]; children: ReactNode }) {
  const user = useAuthStore((s) => s.user);
  const location = useLocation();

  if (user && !allow.includes(user.role)) {
    return <Navigate to="/403" state={{ from: location.pathname }} replace />;
  }
  return <>{children}</>;
}
