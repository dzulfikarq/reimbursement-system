import { forwardRef, type ReactNode } from "react";

interface FormFieldProps {
  label: string;
  htmlFor?: string;
  required?: boolean;
  error?: string;
  hint?: string;
  children: ReactNode;
}

// Label + control + inline error wrapper (docs/05 form UX requirement).
const FormField = forwardRef<HTMLDivElement, FormFieldProps>(
  ({ label, htmlFor, required, error, hint, children }, ref) => {
    return (
      <div ref={ref}>
        <label
          htmlFor={htmlFor}
          className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-400"
        >
          {label}
          {required && <span className="ml-0.5 text-error-500">*</span>}
        </label>
        {children}
        {(error || hint) && (
          <p
            className={`mt-1.5 text-xs ${
              error ? "text-error-500" : "text-gray-500"
            }`}
          >
            {error ?? hint}
          </p>
        )}
      </div>
    );
  },
);

FormField.displayName = "FormField";
export default FormField;
