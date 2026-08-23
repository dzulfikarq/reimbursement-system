import type { TextareaHTMLAttributes } from "react";
import { forwardRef } from "react";

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  error?: boolean;
  hint?: string;
}

// TailAdmin TextArea styling; forwards ref for react-hook-form.
const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className = "", rows = 4, error, hint, ...props }, ref) => {
    const inputClasses = `w-full rounded-lg border px-4 py-2.5 text-sm shadow-theme-xs transition focus:outline-hidden focus:ring-3 ${
      error
        ? "border-error-500 text-error-800 focus:ring-error-500/10 dark:border-error-500 dark:text-error-400"
        : "border-gray-300 bg-transparent text-gray-800 focus:border-brand-300 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
    } ${className}`;

    return (
      <div className="relative">
        <textarea ref={ref} rows={rows} className={inputClasses} {...props} />
        {hint && (
          <p
            className={`mt-1.5 text-xs ${
              error ? "text-error-500" : "text-gray-500"
            }`}
          >
            {hint}
          </p>
        )}
      </div>
    );
  },
);

Textarea.displayName = "Textarea";
export default Textarea;
