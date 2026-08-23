import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useEffect } from "react";
import { useAuthStore } from "../../stores/auth";
import { FullScreenLoader } from "../ui/Loading";

// UX guard only — backend stays source of truth (AGENTS.md).
export default function ProtectedRoute() {
  const { user, status, fetchSession } = useAuthStore();
  const location = useLocation();

  useEffect(() => {
    if (status === "idle") void fetchSession();
  }, [status, fetchSession]);

  if (status !== "ready") return <FullScreenLoader />;
  if (!user) return <Navigate to="/login" replace state={{ from: location }} />;
  return <Outlet />;
}
