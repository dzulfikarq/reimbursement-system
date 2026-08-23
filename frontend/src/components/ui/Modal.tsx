import { useEffect, type ReactNode } from "react";
import { X } from "lucide-react";

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: ReactNode;
  title?: string;
  className?: string;
  showCloseButton?: boolean;
}

// Ported from TailAdmin Modal (escape close, body scroll lock, backdrop).
export default function Modal({
  isOpen,
  onClose,
  children,
  title,
  className = "max-w-lg",
  showCloseButton = true,
}: ModalProps) {
  useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
      document.body.style.overflow = "hidden";
    }
    return () => {
      document.removeEventListener("keydown", handleEscape);
      document.body.style.overflow = "";
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[99999] flex items-center justify-center overflow-y-auto p-4">
      <div
        className="fixed inset-0 h-full w-full bg-gray-400/50 backdrop-blur-[32px]"
        onClick={onClose}
      />
      <div
        role="dialog"
        aria-modal="true"
        className={`relative w-full rounded-3xl bg-white p-6 dark:bg-gray-900 lg:p-8 ${className}`}
        onClick={(e) => e.stopPropagation()}
      >
        {(title || showCloseButton) && (
          <div className="mb-5 flex items-start justify-between gap-3">
            {title && (
              <h3 className="text-lg font-medium text-gray-800 dark:text-white/90">
                {title}
              </h3>
            )}
            {showCloseButton && (
              <button
                onClick={onClose}
                aria-label="Close"
                className="flex size-9 items-center justify-center rounded-full bg-gray-100 text-gray-400 transition-colors hover:bg-gray-200 hover:text-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-white"
              >
                <X className="size-5" />
              </button>
            )}
          </div>
        )}
        {children}
      </div>
    </div>
  );
}
