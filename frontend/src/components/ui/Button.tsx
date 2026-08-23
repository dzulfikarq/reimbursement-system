import type { ReactNode } from "react";
import { Loader2 } from "lucide-react";

interface ButtonProps {
  children: ReactNode;
  size?: "xs" | "sm" | "md";
  variant?: "primary" | "outline" | "danger" | "ghost" | "success";
  startIcon?: ReactNode;
  endIcon?: ReactNode;
  loading?: boolean;
  disabled?: boolean;
  type?: "button" | "submit" | "reset";
  onClick?: () => void;
  className?: string;
}

// Ported from TailAdmin Button; adds danger/ghost/success + loading state.
export default function Button({
  children,
  size = "md",
  variant = "primary",
  startIcon,
  endIcon,
  loading = false,
  disabled = false,
  type = "button",
  onClick,
  className = "",
}: ButtonProps) {
  const sizeClasses = {
    xs: "px-3 py-2 text-xs",
    sm: "px-4 py-3 text-sm",
    md: "px-5 py-3.5 text-sm",
  };

  const variantClasses = {
    primary:
      "bg-brand-500 text-white shadow-theme-xs hover:bg-brand-600 disabled:bg-brand-300",
    outline:
      "bg-white text-gray-700 ring-1 ring-inset ring-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-400 dark:ring-gray-700 dark:hover:bg-white/[0.03] dark:hover:text-gray-300",
    danger:
      "bg-error-500 text-white shadow-theme-xs hover:bg-error-600 disabled:bg-error-300",
    success:
      "bg-success-600 text-white shadow-theme-xs hover:bg-success-700 disabled:bg-success-400",
    ghost:
      "text-gray-700 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-white/[0.05] dark:hover:text-gray-300",
  };

  return (
    <button
      type={type}
      className={`inline-flex items-center justify-center gap-2 rounded-lg font-medium transition ${className} ${
        sizeClasses[size]
      } ${variantClasses[variant]} ${
        disabled || loading ? "cursor-not-allowed opacity-50" : ""
      }`}
      onClick={onClick}
      disabled={disabled || loading}
    >
      {loading ? (
        <Loader2 className="size-4 animate-spin" />
      ) : (
        startIcon && <span className="flex items-center">{startIcon}</span>
      )}
      {children}
      {!loading && endIcon && <span className="flex items-center">{endIcon}</span>}
    </button>
  );
}
