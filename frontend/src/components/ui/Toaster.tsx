import { X, CheckCircle2, AlertCircle, Info } from "lucide-react";
import { useToastStore, type ToastType } from "../../stores/toast";

const ICONS: Record<ToastType, typeof Info> = {
  success: CheckCircle2,
  error: AlertCircle,
  info: Info,
};

const COLORS: Record<ToastType, string> = {
  success: "text-success-600 dark:text-success-400",
  error: "text-error-600 dark:text-error-400",
  info: "text-blue-light-500",
};

// Global toaster — mount once in AppLayout (docs/05).
export default function Toaster() {
  const { toasts, dismiss } = useToastStore();
  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-5 right-5 z-[100] flex w-full max-w-sm flex-col gap-2">
      {toasts.map((t) => {
        const Icon = ICONS[t.type];
        return (
          <div
            key={t.id}
            className="flex items-start gap-3 rounded-xl border border-gray-200 bg-white px-4 py-3 shadow-theme-lg dark:border-gray-800 dark:bg-gray-900"
          >
            <Icon className={`mt-0.5 size-5 shrink-0 ${COLORS[t.type]}`} />
            <p className="flex-1 text-sm text-gray-700 dark:text-gray-300">{t.message}</p>
            <button
              onClick={() => dismiss(t.id)}
              className="text-gray-400 transition hover:text-gray-600 dark:hover:text-gray-200"
              aria-label="Dismiss"
            >
              <X className="size-4" />
            </button>
          </div>
        );
      })}
    </div>
  );
}
