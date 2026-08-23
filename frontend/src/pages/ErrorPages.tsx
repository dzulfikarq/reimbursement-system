import { Link, useLocation } from "react-router-dom";
import { ShieldX } from "lucide-react";
import Button from "../components/ui/Button";

export function ForbiddenPage() {
  const location = useLocation();
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center text-center">
      <ShieldX className="size-14 text-error-500" />
      <h1 className="mt-4 text-2xl font-semibold text-gray-800 dark:text-white/90">403 — Forbidden</h1>
      <p className="mt-1 max-w-sm text-sm text-gray-500 dark:text-gray-400">
        You don't have permission to open <code className="text-xs">{location.pathname}</code>.
      </p>
      <Link to="/" className="mt-5">
        <Button variant="outline">Back to Dashboard</Button>
      </Link>
    </div>
  );
}

export function NotFoundPage() {
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center text-center">
      <h1 className="text-2xl font-semibold text-gray-800 dark:text-white/90">404 — Page not found</h1>
      <p className="mt-1 max-w-sm text-sm text-gray-500 dark:text-gray-400">
        The page you're looking for doesn't exist.
      </p>
      <Link to="/" className="mt-5">
        <Button variant="outline">Back to Dashboard</Button>
      </Link>
    </div>
  );
}
