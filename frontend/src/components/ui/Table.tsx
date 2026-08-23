import type { ReactNode } from "react";

interface TableProps {
  children: ReactNode;
  className?: string;
}
interface TableHeaderProps {
  children: ReactNode;
  className?: string;
}
interface TableBodyProps {
  children: ReactNode;
  className?: string;
}
interface TableRowProps {
  children?: ReactNode;
  className?: string;
  onClick?: () => void;
}
interface TableCellProps {
  children?: ReactNode;
  isHeader?: boolean;
  className?: string;
  onClick?: () => void;
}

// Ported from TailAdmin Table primitives. Header/row styling per TailAdmin
// BasicTables example.
export function Table({ children, className = "" }: TableProps) {
  return (
    <div
      className={`overflow-hidden rounded-2xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03] ${className}`}
    >
      <div className="max-w-full overflow-x-auto custom-scrollbar">
        <table className="min-w-full">{children}</table>
      </div>
    </div>
  );
}

export function TableHeader({ children, className = "" }: TableHeaderProps) {
  return (
    <thead
      className={`border-b border-gray-100 bg-gray-50 dark:border-white/[0.05] dark:bg-gray-900 ${className}`}
    >
      {children}
    </thead>
  );
}

export function TableBody({ children, className = "" }: TableBodyProps) {
  return <tbody className={className}>{children}</tbody>;
}

export function TableRow({ children, className = "", onClick }: TableRowProps) {
  return (
    <tr
      onClick={onClick}
      className={`border-b border-gray-100 last:border-0 hover:bg-gray-50/60 dark:border-white/[0.05] dark:hover:bg-white/[0.02] ${className}`}
    >
      {children}
    </tr>
  );
}

export function TableCell({
  children,
  isHeader = false,
  className = "",
  onClick,
}: TableCellProps) {
  const CellTag = isHeader ? "th" : "td";
  return (
    <CellTag
      onClick={onClick}
      className={`px-5 py-3.5 ${
        isHeader
          ? "text-left text-theme-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400"
          : "text-sm text-gray-700 dark:text-gray-400"
      } ${className}`}
    >
      {children}
    </CellTag>
  );
}
