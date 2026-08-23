import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";
import { Inbox } from "lucide-react";

interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: ReactNode;
}

export default function EmptyState({
  icon: Icon = Inbox,
  title,
  description,
  action,
}: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-gray-300 bg-white px-6 py-16 text-center dark:border-gray-700 dark:bg-white/[0.03]">
      <div className="flex size-14 items-center justify-center rounded-full bg-gray-100 text-gray-400 dark:bg-white/5">
        <Icon className="size-7" />
      </div>
      <h3 className="mt-4 text-base font-medium text-gray-800 dark:text-white/90">
        {title}
      </h3>
      {description && (
        <p className="mt-1 max-w-sm text-sm text-gray-500 dark:text-gray-400">
          {description}
        </p>
      )}
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}

// Error panel with retry for query failures.
export function ErrorState({
  message,
  onRetry,
}: {
  message?: string;
  onRetry?: () => void;
}) {
  return (
    <div className="rounded-2xl border border-error-200 bg-error-50 px-6 py-10 text-center dark:border-error-500/30 dark:bg-error-500/10">
      <h3 className="text-base font-medium text-error-700 dark:text-error-400">
        Something went wrong
      </h3>
      <p className="mt-1 text-sm text-error-600/80 dark:text-error-400/80">
        {message ?? "Failed to load data. Please try again."}
      </p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-4 rounded-lg bg-error-500 px-4 py-2 text-sm font-medium text-white transition hover:bg-error-600"
        >
          Retry
        </button>
      )}
    </div>
  );
}
