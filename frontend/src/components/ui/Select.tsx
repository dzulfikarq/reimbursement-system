import type { SelectHTMLAttributes } from "react";
import { forwardRef } from "react";

export interface SelectOption {
  value: string;
  label: string;
}

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  options: SelectOption[];
  placeholder?: string;
  error?: boolean;
  hint?: string;
}

// Native select styled like TailAdmin's custom Select (no headless dep needed).
const Select = forwardRef<HTMLSelectElement, SelectProps>(
  (
    {
      options,
      placeholder,
      error,
      hint,
      className = "",
      defaultValue = "",
      ...props
    },
    ref,
  ) => {
    const baseClasses = `h-11 w-full appearance-none rounded-lg border bg-transparent px-4 py-2.5 pr-11 text-sm shadow-theme-xs transition focus:outline-hidden focus:ring-3 ${
      error
        ? "border-error-500 text-error-800 focus:ring-error-500/10 dark:border-error-500 dark:text-error-400"
        : "border-gray-300 focus:border-brand-300 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:focus:border-brand-800"
    } ${className}`;

    return (
      <div className="relative">
        <select
          ref={ref}
          defaultValue={defaultValue}
          className={`${baseClasses} text-gray-800 dark:text-white/90`}
          {...props}
        >
          {placeholder && (
            <option
              value=""
              disabled
              className="text-gray-700 dark:bg-gray-900 dark:text-gray-400"
            >
              {placeholder}
            </option>
          )}
          {options.map((o) => (
            <option
              key={o.value}
              value={o.value}
              className="text-gray-700 dark:bg-gray-900 dark:text-gray-400"
            >
              {o.label}
            </option>
          ))}
        </select>
        <svg
          className="pointer-events-none absolute right-4 top-1/2 -translate-y-1/2 text-gray-400"
          width="20"
          height="20"
          viewBox="0 0 20 20"
          fill="none"
        >
          <path
            d="M4.79175 7.39583L10.0001 12.6042L15.2084 7.39583"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
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

Select.displayName = "Select";
export default Select;
