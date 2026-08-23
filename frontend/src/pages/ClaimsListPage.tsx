import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Plus, Search, ReceiptText, ChevronLeft, ChevronRight, Eye } from "lucide-react";
import Input from "../components/ui/Input";
import Select from "../components/ui/Select";
import Badge, { statusBadgeColor } from "../components/ui/Badge";
import Button from "../components/ui/Button";
import { Table, TableHeader, TableBody, TableRow, TableCell } from "../components/ui/Table";
import EmptyState, { ErrorState } from "../components/ui/EmptyState";
import { useClaims } from "../lib/claims";
import { useCategories } from "../lib/admin";

const STATUSES = ["", "draft", "submitted", "approved", "rejected", "paid", "cancelled"];

export function fmtIDR(v: string | number): string {
  const n = typeof v === "string" ? parseFloat(v) : v;
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n || 0);
}

export default function ClaimsListPage() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");

  const categories = useCategories();
  const claims = useClaims({
    page,
    limit: 10,
    status: status || undefined,
    category_id: categoryId || undefined,
    date_from: dateFrom || undefined,
    date_to: dateTo || undefined,
    search: search || undefined,
    sort: "created_at",
    order: "desc",
  });

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-title-md font-semibold text-gray-800 dark:text-white/90">
          Reimbursements
        </h1>
        <Link to="/reimbursements/new">
          <Button size="sm" startIcon={<Plus className="size-4" />}>New Claim</Button>
        </Link>
      </div>

      {/* Filters */}
      <div className="grid grid-cols-2 gap-3 rounded-2xl border border-gray-200 bg-white p-4 sm:grid-cols-3 lg:grid-cols-6 dark:border-gray-800 dark:bg-white/[0.03]">
        <form
          className="relative col-span-2 lg:col-span-2"
          onSubmit={(e) => {
            e.preventDefault();
            setPage(1);
            setSearch(searchInput);
          }}
        >
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-gray-400" />
          <Input
            placeholder="Search title…"
            className="!pl-9"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
          />
        </form>
        <Select
          value={status}
          onChange={(e) => { setPage(1); setStatus(e.target.value); }}
          options={[
            { value: "", label: "All statuses" },
            ...STATUSES.filter(Boolean).map((s) => ({ value: s, label: s })),
          ]}
        />
        <Select
          value={categoryId}
          onChange={(e) => { setPage(1); setCategoryId(e.target.value); }}
          options={[
            { value: "", label: "All categories" },
            ...(categories.data ?? []).map((c) => ({ value: c.id, label: c.name })),
          ]}
        />
        <Input type="date" value={dateFrom} onChange={(e) => { setPage(1); setDateFrom(e.target.value); }} />
        <Input type="date" value={dateTo} onChange={(e) => { setPage(1); setDateTo(e.target.value); }} />
      </div>

      {claims.isLoading ? (
        <div className="space-y-2">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="h-14 animate-pulse rounded-xl bg-gray-100 dark:bg-white/5" />
          ))}
        </div>
      ) : claims.isError ? (
        <ErrorState onRetry={() => claims.refetch()} />
      ) : claims.data!.items.length === 0 ? (
        <EmptyState
          icon={ReceiptText}
          title="No reimbursements found"
          description="Nothing matches these filters. Try clearing them or create a new claim."
          action={
            <Link to="/reimbursements/new">
              <Button size="sm">Create claim</Button>
            </Link>
          }
        />
      ) : (
        <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableCell isHeader>Title</TableCell>
                <TableCell isHeader>Category</TableCell>
                <TableCell isHeader>Expense date</TableCell>
                <TableCell isHeader>Status</TableCell>
                <TableCell isHeader className="!text-right">Amount</TableCell>
                <TableCell isHeader />
              </TableRow>
            </TableHeader>
            <TableBody>
              {claims.data!.items.map((c) => (
                <TableRow
                  key={c.id}
                  className="cursor-pointer"
                  onClick={() => navigate(`/reimbursements/${c.id}`)}
                >
                  <TableCell className="max-w-[240px] truncate font-medium text-gray-800 dark:text-white/90">
                    {c.title}
                    {c.employee_name && (
                      <span className="block text-theme-xs font-normal text-gray-400">{c.employee_name}</span>
                    )}
                  </TableCell>
                  <TableCell>{c.category_name}</TableCell>
                  <TableCell>{c.expense_date}</TableCell>
                  <TableCell>
                    <Badge color={statusBadgeColor[c.status] ?? "light"}>{c.status}</Badge>
                  </TableCell>
                  <TableCell className="!text-right font-medium">{fmtIDR(c.amount)}</TableCell>
                  <TableCell className="!text-right">
                    <Eye className="ml-auto size-4 text-gray-400" />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {/* Pagination */}
          {claims.data!.meta.total_pages > 1 && (
            <div className="flex items-center justify-between text-sm text-gray-500">
              <span>
                Page {claims.data!.meta.page} of {claims.data!.meta.total_pages} ({claims.data!.meta.total} claims)
              </span>
              <div className="flex gap-2">
                <Button variant="outline" size="xs" disabled={page <= 1} onClick={() => setPage(page - 1)}>
                  <ChevronLeft className="size-4" /> Prev
                </Button>
                <Button
                  variant="outline"
                  size="xs"
                  disabled={page >= claims.data!.meta.total_pages}
                  onClick={() => setPage(page + 1)}
                >
                  Next <ChevronRight className="size-4" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
