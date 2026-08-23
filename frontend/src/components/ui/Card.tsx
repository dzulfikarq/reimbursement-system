import type { ReactNode } from "react";

interface CardProps {
  title?: string;
  desc?: string;
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
}

// Ported from TailAdmin ComponentCard.
export default function Card({
  title,
  desc,
  children,
  className = "",
  bodyClassName = "",
}: CardProps) {
  return (
    <div
      className={`rounded-2xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03] ${className}`}
    >
      {title && (
        <div className="px-6 pt-5">
          <h3 className="text-base font-medium text-gray-800 dark:text-white/90">
            {title}
          </h3>
          {desc && (
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{desc}</p>
          )}
        </div>
      )}
      <div
        className={`p-4 sm:p-6 ${title ? "" : ""} ${bodyClassName}`}
      >
        <div className="space-y-6">{children}</div>
      </div>
    </div>
  );
}
