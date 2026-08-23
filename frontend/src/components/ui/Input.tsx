import type { InputHTMLAttributes } from "react";
import { forwardRef } from "react";

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  error?: boolean;
  success?: boolean;
  hint?: string;
}

// Ported from TailAdmin Input (state styles + hint). Forwards ref so it works
// with react-hook-form register().
const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className = "", disabled, error, success, hint, ...props }, ref) => {
    let inputClasses = `h-11 w-full appearance-none rounded-lg border px-4 py-2.5 text-sm shadow-theme-xs transition focus:outline-hidden focus:ring-3 ${className}`;

    if (disabled) {
      inputClasses +=
        " cursor-not-allowed border-gray-300 text-gray-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400";
    } else if (error) {
      inputClasses +=
        " border-error-500 text-error-800 focus:ring-error-500/10 dark:border-error-500 dark:text-error-400";
    } else if (success) {
      inputClasses +=
        " border-success-400 text-success-500 focus:border-success-300 focus:ring-success-500/10 dark:border-success-500 dark:text-success-400";
    } else {
      inputClasses +=
        " border-gray-300 bg-transparent text-gray-800 focus:border-brand-300 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800";
    }

    return (
      <div className="relative">
        <input
          ref={ref}
          disabled={disabled}
          className={inputClasses}
          {...props}
        />
        {hint && (
          <p
            className={`mt-1.5 text-xs ${
              error
                ? "text-error-500"
                : success
                  ? "text-success-500"
                  : "text-gray-500"
            }`}
          >
            {hint}
          </p>
        )}
      </div>
    );
  },
);

Input.displayName = "Input";
export default Input;
