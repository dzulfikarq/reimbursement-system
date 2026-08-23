import { Link } from "react-router-dom";
import {
  Clock,
  Banknote,
  TrendingUp,
  AlertTriangle,
  Plus,
  ClipboardCheck,
} from "lucide-react";
import Card from "../components/ui/Card";
import Badge from "../components/ui/Badge";
import Button from "../components/ui/Button";
import { Table, TableHeader, TableBody, TableRow, TableCell } from "../components/ui/Table";
import { useAuthStore } from "../stores/auth";
import {
  useDashboardSummary,
  useMonthlyTrend,
  useCategoryBreakdown,
} from "../lib/claims";

function fmtIDR(v: string | number): string {
  const n = typeof v === "string" ? parseFloat(v) : v;
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n || 0);
}

const ROLE_ACTIONS: Record<string, { to: string; label: string }> = {
  employee: { to: "/reimbursements/new", label: "New Claim" },
  manager: { to: "/approvals", label: "Go to Approvals" },
  finance: { to: "/payments", label: "Payment Queue" },
};

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  const role = user?.role ?? "employee";
  const summary = useDashboardSummary();
  const trend = useMonthlyTrend(6);
  const breakdown = useCategoryBreakdown();

  const action = ROLE_ACTIONS[role];

  if (summary.isLoading) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="h-28 animate-pulse rounded-2xl bg-gray-100 dark:bg-white/5" />
        ))}
      </div>
    );
  }

  if (summary.isError) {
    return (
      <div className="rounded-2xl border border-error-200 bg-error-50 px-6 py-10 text-center dark:border-error-500/30 dark:bg-error-500/10">
        <p className="text-sm text-error-600 dark:text-error-400">Failed to load dashboard.</p>
        <Button variant="outline" className="mt-4" onClick={() => summary.refetch()}>Retry</Button>
      </div>
    );
  }

  const s = summary.data;
  const maxTrend = Math.max(...(trend.data ?? []).map((t) => parseFloat(t.total) || 0), 1);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-title-md font-semibold text-gray-800 dark:text-white/90">Dashboard</h1>
          <p className="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
            Welcome back, {user?.name}.
          </p>
        </div>
        {action && (
          <Link to={action.to}>
            <Button startIcon={<Plus className="size-4" />}>{action.label}</Button>
          </Link>
        )}
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <Card>
          <div className="flex items-center gap-4 p-5">
            <div className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-warning-50 text-warning-500 dark:bg-warning-500/10">
              <Clock className="size-6" />
            </div>
            <div>
              <p className="text-theme-xs font-medium uppercase tracking-wide text-gray-500">Pending claims</p>
              <p className="mt-0.5 text-2xl font-bold text-gray-800 dark:text-white/90">{s?.pending_count ?? 0}</p>
            </div>
          </div>
        </Card>
        <Card>
          <div className="flex items-center gap-4 p-5">
            <div className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-brand-50 text-brand-500 dark:bg-brand-500/10">
              <Banknote className="size-6" />
            </div>
            <div>
              <p className="text-theme-xs font-medium uppercase tracking-wide text-gray-500">This month</p>
              <p className="mt-0.5 text-2xl font-bold text-gray-800 dark:text-white/90">{fmtIDR(s?.monthly_total ?? "0")}</p>
            </div>
          </div>
        </Card>
        <Card>
          <div className="flex items-center gap-4 p-5">
            <div className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-success-50 text-success-600 dark:bg-success-500/10">
              <TrendingUp className="size-6" />
            </div>
            <div>
              <p className="text-theme-xs font-medium uppercase tracking-wide text-gray-500">Approval rate</p>
              <p className="mt-0.5 text-2xl font-bold text-gray-800 dark:text-white/90">
                {s?.approval_rate == null ? "—" : `${Math.round(s.approval_rate)}%`}
              </p>
            </div>
          </div>
        </Card>
      </div>

      {/* Budget warnings (manager+) */}
      {(role === "manager" || role === "finance" || role === "admin") && (s?.budget_usage?.length ?? 0) > 0 && (
        <Card title="Department Budget Usage" desc="Current month spend vs monthly budget.">
          <div className="space-y-3 px-5 pb-5 pt-1">
            {s!.budget_usage.map((b) => {
              const warn = b.used_percent >= 80;
              return (
                <div key={b.department_id}>
                  <div className="mb-1 flex items-center justify-between text-sm">
                    <span className={`inline-flex items-center gap-1.5 font-medium ${warn ? "text-error-600" : "text-gray-700 dark:text-gray-300"}`}>
                      {warn && <AlertTriangle className="size-4" />}
                      {b.department_name}
                    </span>
                    <span className="text-gray-500">
                      {fmtIDR(b.monthly_spend)} / {b.monthly_budget ? fmtIDR(b.monthly_budget) : "no budget"}
                    </span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-white/5">
                    <div
                      className={`h-full rounded-full transition-all ${warn ? "bg-error-500" : "bg-brand-500"}`}
                      style={{ width: `${Math.min(b.used_percent, 100)}%` }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Trend */}
        <Card title="Monthly Trend" desc="Last 6 months of submitted claim value.">
          <div className="px-5 pb-5 pt-2">
            {trend.isLoading ? (
              <div className="h-40 animate-pulse rounded-xl bg-gray-100 dark:bg-white/5" />
            ) : (trend.data?.length ?? 0) === 0 ? (
              <p className="py-10 text-center text-sm text-gray-400">No data yet.</p>
            ) : (
              <div className="flex h-40 items-end gap-3">
                {trend.data!.map((t) => {
                  const h = Math.max(((parseFloat(t.total) || 0) / maxTrend) * 100, 2);
                  return (
                    <div key={t.month} className="group flex flex-1 flex-col items-center gap-1.5">
                      <span className="text-theme-xs text-gray-400 opacity-0 transition group-hover:opacity-100">
                        {fmtIDR(t.total)}
                      </span>
                      <div
                        className="w-full rounded-t-lg bg-brand-500/80 transition group-hover:bg-brand-500"
                        style={{ height: `${h}%` }}
                      />
                      <span className="text-theme-xs text-gray-500">{t.month.slice(5)}</span>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </Card>

        {/* Category breakdown */}
        <Card title="Top Categories" desc="This month by total amount.">
          <div className="px-5 pb-5 pt-1">
            {breakdown.isLoading ? (
              <div className="space-y-2 py-2">
                {[...Array(4)].map((_, i) => (
                  <div key={i} className="h-8 animate-pulse rounded-lg bg-gray-100 dark:bg-white/5" />
                ))}
              </div>
            ) : (breakdown.data?.length ?? 0) === 0 ? (
              <p className="py-10 text-center text-sm text-gray-400">No claims this month.</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableCell isHeader>Category</TableCell>
                    <TableCell isHeader className="!text-right">Claims</TableCell>
                    <TableCell isHeader className="!text-right">Total</TableCell>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {breakdown.data!.slice(0, 5).map((c) => (
                    <TableRow key={c.category_id}>
                      <TableCell>{c.category_name}</TableCell>
                      <TableCell className="!text-right">{c.claim_count}</TableCell>
                      <TableCell className="!text-right font-medium">{fmtIDR(c.total)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </div>
        </Card>
      </div>

      {role !== "employee" && (
        <Link
          to="/approvals"
          className="flex items-center gap-3 rounded-2xl border border-dashed border-brand-200 bg-brand-50/50 px-5 py-4 text-sm font-medium text-brand-600 transition hover:bg-brand-50 dark:border-brand-500/20 dark:bg-brand-500/5 dark:hover:bg-brand-500/10"
        >
          <ClipboardCheck className="size-5" />
          You have work waiting in the approval queue.
          <Badge color="warning" size="sm">{s?.pending_count ?? 0} pending</Badge>
        </Link>
      )}
    </div>
  );
}
