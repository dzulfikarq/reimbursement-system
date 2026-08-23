import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ChevronLeft, ChevronRight, ClipboardCheck, Banknote } from "lucide-react";
import Badge, { statusBadgeColor } from "../components/ui/Badge";
import Button from "../components/ui/Button";
import ConfirmDialog from "../components/ui/ConfirmDialog";
import Modal from "../components/ui/Modal";
import FormField from "../components/ui/FormField";
import Textarea from "../components/ui/Textarea";
import { Table, TableHeader, TableBody, TableRow, TableCell } from "../components/ui/Table";
import EmptyState, { ErrorState } from "../components/ui/EmptyState";
import { fmtIDR } from "./ClaimsListPage";
import { toast } from "../stores/toast";
import { api } from "../lib/api";
import { useClaims, type Claim } from "../lib/claims";

interface QueueProps {
  mode: "approvals" | "payments";
}

// Approvals: SUBMITTED claims (manager sees dept scope; finance/admin all).
// Payments: APPROVED claims awaiting finance disbursement.
export default function ClaimsQueuePage({ mode }: QueueProps) {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const status = mode === "approvals" ? "submitted" : "approved";
  const list = useClaims({ page, limit: 10, status, sort: "created_at", order: "asc" });

  const [confirmTarget, setConfirmTarget] = useState<{ claim: Claim; action: "approve" | "pay" | "reject" } | null>(null);
  const [rejectNote, setRejectNote] = useState("");
  const [rejectOpen, setRejectOpen] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);

  function act(action: "approve" | "pay" | "reject", claim: Claim, note?: string) {
    setBusyId(claim.id);
    // Direct call keeps the table lean; server still validates turn & role.
    api
      .post(`/reimbursements/${claim.id}/${action}`, note ? { note } : {})
      .then(() => {
        toast.success(action === "approve" ? "Approved" : action === "pay" ? "Marked as paid" : "Rejected");
        setConfirmTarget(null);
        setRejectOpen(false);
        setRejectNote("");
        list.refetch();
      })
      .catch((e) => toast.error(e?.response?.data?.error?.message ?? "Action failed"))
      .finally(() => setBusyId(null));
  }

  return (
    <div className="space-y-5">
      <h1 className="text-title-md font-semibold text-gray-800 dark:text-white/90">
        {mode === "approvals" ? "Approval queue" : "Payment queue"}
      </h1>

      {list.isLoading ? (
        <div className="space-y-2">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="h-14 animate-pulse rounded-xl bg-gray-100 dark:bg-white/5" />
          ))}
        </div>
      ) : list.isError ? (
        <ErrorState onRetry={() => list.refetch()} />
      ) : list.data!.items.length === 0 ? (
        <EmptyState
          icon={mode === "approvals" ? ClipboardCheck : Banknote}
          title={mode === "approvals" ? "Nothing waiting for approval" : "No claims ready for payment"}
          description={
            mode === "approvals"
              ? "Submitted claims will appear here when it's your turn to review."
              : "Approved claims appear here until Finance marks them paid."
          }
        />
      ) : (
        <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableCell isHeader>Claim</TableCell>
                <TableCell isHeader>Employee</TableCell>
                <TableCell isHeader>Expense date</TableCell>
                <TableCell isHeader className="!text-right">Amount</TableCell>
                <TableCell isHeader className="!text-right">Actions</TableCell>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.data!.items.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="cursor-pointer font-medium text-gray-800 hover:text-brand-500 dark:text-white/90" onClick={() => navigate(`/reimbursements/${c.id}`)}>
                    {c.title}
                    <span className="ml-2"><Badge size="sm" color={statusBadgeColor[c.status] ?? "warning"}>{c.status}</Badge></span>
                  </TableCell>
                  <TableCell>{c.employee_name}</TableCell>
                  <TableCell>{c.expense_date}</TableCell>
                  <TableCell className="!text-right font-medium">{fmtIDR(c.amount)}</TableCell>
                  <TableCell className="!text-right">
                    <div className="flex justify-end gap-1.5">
                      {mode === "approvals" && (
                        <>
                          <Button size="xs" variant="success" loading={busyId === c.id} onClick={() => setConfirmTarget({ claim: c, action: "approve" })}>
                            Approve
                          </Button>
                          <Button size="xs" variant="outline" onClick={() => setConfirmTarget({ claim: c, action: "reject" })}>
                            Reject
                          </Button>
                        </>
                      )}
                      {mode === "payments" && (
                        <Button size="xs" variant="success" loading={busyId === c.id} onClick={() => setConfirmTarget({ claim: c, action: "pay" })}>
                          Mark paid
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {list.data!.meta.total_pages > 1 && (
            <div className="flex items-center justify-between text-sm text-gray-500">
              <span>Page {list.data!.meta.page} of {list.data!.meta.total_pages}</span>
              <div className="flex gap-2">
                <Button variant="outline" size="xs" disabled={page <= 1} onClick={() => setPage(page - 1)}>
                  <ChevronLeft className="size-4" /> Prev
                </Button>
                <Button variant="outline" size="xs" disabled={page >= list.data!.meta.total_pages} onClick={() => setPage(page + 1)}>
                  Next <ChevronRight className="size-4" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      {/* Approve / pay confirmation */}
      <ConfirmDialog
        isOpen={confirmTarget?.action === "approve" || confirmTarget?.action === "pay"}
        onClose={() => setConfirmTarget(null)}
        onConfirm={() => confirmTarget && act(confirmTarget.action, confirmTarget.claim)}
        title={confirmTarget?.action === "approve" ? "Approve claim?" : "Mark as paid?"}
        message={
          confirmTarget
            ? `"${confirmTarget.claim.title}" (${fmtIDR(confirmTarget.claim.amount)})`
            : ""
        }
        tone={confirmTarget?.action === "pay" ? "success" : "primary"}
        loading={busyId !== null}
      />

      {/* Reject note */}
      <Modal isOpen={rejectOpen || confirmTarget?.action === "reject"} onClose={() => { setRejectOpen(false); setConfirmTarget(null); }} title="Reject claim">
        <form
          className="space-y-4 p-5"
          onSubmit={(e) => {
            e.preventDefault();
            if (!confirmTarget) return;
            if (rejectNote.trim().length < 3) return toast.error("Note must be at least 3 characters");
            act("reject", confirmTarget.claim, rejectNote.trim());
          }}
        >
          <FormField label="Reason" required hint="Required by the backend — rejection without a note is rejected.">
            <Textarea rows={3} value={rejectNote} onChange={(e) => setRejectNote(e.target.value)} placeholder="Why is this claim rejected?" autoFocus />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" type="button" onClick={() => { setRejectOpen(false); setConfirmTarget(null); }}>Cancel</Button>
            <Button variant="danger" size="sm" type="submit" loading={busyId !== null}>Reject</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
