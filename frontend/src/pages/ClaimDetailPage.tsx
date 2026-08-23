import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  ArrowLeft,
  Paperclip,
  Download,
  Trash2,
  UploadCloud,
  FileText,
} from "lucide-react";
import Card from "../components/ui/Card";
import Badge, { statusBadgeColor } from "../components/ui/Badge";
import Button from "../components/ui/Button";
import ConfirmDialog from "../components/ui/ConfirmDialog";
import Modal from "../components/ui/Modal";
import FormField from "../components/ui/FormField";
import Textarea from "../components/ui/Textarea";
import { Table, TableHeader, TableBody, TableRow, TableCell } from "../components/ui/Table";
import { useAuthStore } from "../stores/auth";
import { toast, errMessage } from "../stores/toast";
import { fmtIDR } from "./ClaimsListPage";
import { useClaim, useClaimAction, useDeleteClaim, useUploadAttachment, type WorkflowAction } from "../lib/claims";

const MAX_UPLOAD = 5 * 1024 * 1024;
const MIME_OK = ["image/png", "image/jpeg", "image/webp", "application/pdf"];

export default function ClaimDetailPage() {
  const { id } = useParams<{ id: string }>();
  const user = useAuthStore((s) => s.user);
  const claim = useClaim(id);
  const action = useClaimAction(id!);
  const del = useDeleteClaim(id!);
  const upload = useUploadAttachment(id!);

  const [confirm, setConfirm] = useState<WorkflowAction | "delete" | null>(null);
  const [rejectOpen, setRejectOpen] = useState(false);
  const [note, setNote] = useState("");

  if (claim.isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-40 animate-pulse rounded-lg bg-gray-100 dark:bg-white/5" />
        <div className="h-48 animate-pulse rounded-2xl bg-gray-100 dark:bg-white/5" />
        <div className="h-64 animate-pulse rounded-2xl bg-gray-100 dark:bg-white/5" />
      </div>
    );
  }
  if (claim.isError || !claim.data) {
    return (
      <div className="rounded-2xl border border-error-200 bg-error-50 px-6 py-10 text-center dark:border-error-500/30 dark:bg-error-500/10">
        <p className="text-sm text-error-600 dark:text-error-400">Failed to load claim.</p>
        <Button variant="outline" className="mt-4" onClick={() => claim.refetch()}>Retry</Button>
      </div>
    );
  }

  const c = claim.data;
  const isOwner = user?.id === c.employee_id;
  const role = user?.role ?? "";
  // Edit/delete only in DRAFT; cancel only SUBMITTED (server enforces — these
  // buttons are UX affordances mirroring docs/02).
  const canEdit = isOwner && (c.status === "draft" || c.status === "rejected");
  const canDelete = isOwner && c.status === "draft";
  const canSubmit = isOwner && c.status === "draft";
  const canCancel = isOwner && c.status === "submitted";
  const canPay = (role === "finance" || role === "admin") && c.status === "approved";

  function runAct(a: WorkflowAction, n?: string) {
    action.mutate(
      { action: a, note: n },
      { onSuccess: () => setConfirm(null) },
    );
  }

  function onUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0];
    e.target.value = "";
    if (!f) return;
    if (!MIME_OK.includes(f.type)) return toast.error("Only PNG, JPEG, WebP or PDF allowed");
    if (f.size > MAX_UPLOAD) return toast.error("File must be under 5 MB");
    upload.mutate(f);
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <Link to="/reimbursements" className="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-brand-500">
            <ArrowLeft className="size-4" /> Back to list
          </Link>
          <div className="mt-1 flex items-center gap-3">
            <h1 className="text-title-md font-semibold text-gray-800 dark:text-white/90">{c.title}</h1>
            <Badge color={statusBadgeColor[c.status] ?? "light"}>{c.status}</Badge>
          </div>
          {c.employee_name && <p className="text-sm text-gray-500">{c.employee_name} · {c.expense_date}</p>}
        </div>

        {/* Actions */}
        <div className="flex flex-wrap gap-2">
          {canEdit && (
            <Link to={`/reimbursements/${c.id}/edit`}>
              <Button variant="outline" size="sm">Edit</Button>
            </Link>
          )}
          {canSubmit && (
            <Button size="sm" loading={action.isPending} onClick={() => setConfirm("submit")}>Submit</Button>
          )}
          {canCancel && (
            <Button variant="danger" size="sm" loading={action.isPending} onClick={() => setConfirm("cancel")}>Cancel</Button>
          )}
          {(role === "manager" || role === "finance" || role === "admin") && c.status === "submitted" && (
            <>
              <Button variant="success" size="sm" loading={action.isPending} onClick={() => setConfirm("approve")}>Approve</Button>
              <Button variant="danger" size="sm" onClick={() => setRejectOpen(true)}>Reject</Button>
            </>
          )}
          {canPay && (
            <Button variant="success" size="sm" loading={action.isPending} onClick={() => setConfirm("pay")}>Mark as Paid</Button>
          )}
          {canDelete && (
            <Button variant="ghost" size="sm" loading={del.isPending} onClick={() => setConfirm("delete")}>
              <Trash2 className="size-4 text-error-500" />
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Items */}
        <Card title="Expense items" desc={`Total: ${fmtIDR(c.amount)}`} className="lg:col-span-2">
          <Table>
            <TableHeader>
              <TableRow>
                <TableCell isHeader>Description</TableCell>
                <TableCell isHeader className="!text-right">Qty</TableCell>
                <TableCell isHeader className="!text-right">Unit price</TableCell>
                <TableCell isHeader className="!text-right">Total</TableCell>
              </TableRow>
            </TableHeader>
            <TableBody>
              {c.items.map((it) => (
                <TableRow key={it.id}>
                  <TableCell>{it.description}</TableCell>
                  <TableCell className="!text-right">{it.quantity}</TableCell>
                  <TableCell className="!text-right">{fmtIDR(it.unit_price)}</TableCell>
                  <TableCell className="!text-right font-medium">{fmtIDR(it.line_total)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>

        {/* Timeline */}
        <Card title="Approval timeline">
          <ol className="space-y-4 px-5 pb-5 pt-2">
            {(c.approvals ?? []).map((step) => (
              <li key={step.step_number} className="relative pl-6">
                <span
                  className={`absolute left-0 top-0.5 size-3 rounded-full ${
                    step.status === "approved"
                      ? "bg-success-500"
                      : step.status === "rejected"
                        ? "bg-error-500"
                        : "bg-warning-400"
                  }`}
                />
                {step.step_number !== (c.approvals?.length ?? 0) && (
                  <span className="absolute left-[5px] top-4 h-full w-px bg-gray-200 dark:bg-white/10" />
                )}
                <p className="text-sm font-medium capitalize text-gray-800 dark:text-white/90">
                  Step {step.step_number}: {step.approver_role}
                </p>
                <Badge size="sm" color={statusBadgeColor[step.status] === "dark" ? "warning" : statusBadgeColor[step.status] ?? "warning"}>
                  {step.status}
                </Badge>
                {step.note && <p className="mt-1 text-xs italic text-gray-500">"{step.note}"</p>}
              </li>
            ))}
            {c.status === "draft" && (
              <li className="pl-6 text-sm text-gray-400">Not submitted yet — no approval steps.</li>
            )}
            {c.status === "cancelled" && <li className="pl-6 text-sm text-gray-400">Cancelled by owner.</li>}
          </ol>
        </Card>
      </div>

      {/* Attachments */}
      <Card title="Receipts & attachments">
        <div className="px-5 pb-5 pt-1">
          {isOwner && (c.status === "draft" || c.status === "rejected") && (
            <label className="mb-4 flex cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-gray-300 px-6 py-6 text-sm text-gray-500 transition hover:border-brand-400 hover:bg-brand-50/40 dark:border-gray-700 dark:hover:bg-brand-500/5">
              <UploadCloud className="size-6 text-gray-400" />
              {upload.isPending ? "Uploading…" : "Click to upload receipt (PNG/JPEG/WebP/PDF, max 5 MB)"}
              <input type="file" accept=".png,.jpg,.jpeg,.webp,.pdf" className="hidden" onChange={onUpload} disabled={upload.isPending} />
            </label>
          )}

          {c.attachments.length === 0 ? (
            <p className="flex items-center gap-2 py-2 text-sm text-gray-400">
              <Paperclip className="size-4" /> No attachments.
            </p>
          ) : (
            <ul className="divide-y divide-gray-100 dark:divide-white/5">
              {c.attachments.map((a) => (
                <li key={a.id} className="flex items-center gap-3 py-2.5">
                  <FileText className="size-4 shrink-0 text-gray-400" />
                  <span className="flex-1 truncate text-sm text-gray-700 dark:text-gray-300">{a.original_filename}</span>
                  <span className="text-theme-xs text-gray-400">{(a.size_bytes / 1024).toFixed(0)} KB</span>
                  <a href={`${import.meta.env.VITE_API_BASE ?? "/api/v1"}/attachments/${a.id}/download`} target="_blank" rel="noreferrer">
                    <Button variant="outline" size="xs" startIcon={<Download className="size-3.5" />}>Download</Button>
                  </a>
                </li>
              ))}
            </ul>
          )}
        </div>
      </Card>

      {/* Confirmations */}
      <ConfirmDialog
        isOpen={confirm !== null}
        onClose={() => setConfirm(null)}
        onConfirm={() => confirm === "delete" ? del.mutate(undefined as never) : runAct(confirm as WorkflowAction)}
        title={
          confirm === "submit" ? "Submit this claim?" :
          confirm === "cancel" ? "Cancel this claim?" :
          confirm === "approve" ? "Approve this claim?" :
          confirm === "pay" ? "Mark claim as paid?" :
          "Delete draft?"
        }
        message={
          confirm === "submit" ? "Approval steps will be generated based on the amount." :
          confirm === "cancel" ? "The claim will be cancelled and removed from the approval queue." :
          confirm === "approve" ? "You approve the current step of this claim." :
          confirm === "pay" ? "Finance confirms payment has been made to the employee." :
          "This draft and its attachments will be permanently deleted."
        }
        confirmLabel={confirm === "delete" ? "Delete" : confirm === "submit" ? "Submit" : confirm === "cancel" ? "Cancel claim" : confirm === "approve" ? "Approve" : confirm === "pay" ? "Mark paid" : "Confirm"}
        tone={confirm === "approve" || confirm === "pay" ? "success" : confirm === "submit" ? "primary" : "danger"}
        loading={action.isPending || del.isPending}
      />

      {/* Reject note modal */}
      <Modal isOpen={rejectOpen} onClose={() => setRejectOpen(false)} title="Reject claim">
        <form
          className="space-y-4 p-5"
          onSubmit={(e) => {
            e.preventDefault();
            if (note.trim().length < 3) return toast.error(errMessage(new Error("Note must be at least 3 characters")));
            runAct("reject", note.trim());
            setRejectOpen(false);
            setNote("");
          }}
        >
          <FormField label="Reason" required error={note.trim().length > 0 && note.trim().length < 3 ? "Min 3 characters" : ""}>
            <Textarea rows={3} value={note} onChange={(e) => setNote(e.target.value)} placeholder="Why is this claim rejected?" autoFocus />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" type="button" onClick={() => setRejectOpen(false)}>Cancel</Button>
            <Button variant="danger" size="sm" type="submit" loading={action.isPending}>Reject claim</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
