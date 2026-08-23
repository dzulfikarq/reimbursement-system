import { useState } from "react";
import { Plus, KeyRound } from "lucide-react";
import Badge from "../components/ui/Badge";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import Select from "../components/ui/Select";
import FormField from "../components/ui/FormField";
import Modal from "../components/ui/Modal";
import ConfirmDialog from "../components/ui/ConfirmDialog";
import { Table, TableHeader, TableBody, TableRow, TableCell } from "../components/ui/Table";
import EmptyState, { ErrorState } from "../components/ui/EmptyState";
import { fmtIDR } from "./ClaimsListPage";
import { toast } from "../stores/toast";
import { api } from "../lib/api";
import {
  useCategories,
  useDepartments,
  useUsers,
  useSaveCategory,
  useSaveDepartment,
  useDeleteCategory,
  useDeleteDepartment,
  useCreateUser,
  useResetPassword,
  type Category,
  type Department,
  type UserRow,
} from "../lib/admin";

// --- Categories ---
export function AdminCategoriesPage() {
  const categories = useCategories();
  const [editing, setEditing] = useState<Category | "new" | null>(null);
  const [deleting, setDeleting] = useState<Category | null>(null);
  const del = useDeleteCategory();

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="text-title-md font-semibold text-gray-800 dark:text-white/90">Categories</h1>
        <Button size="sm" startIcon={<Plus className="size-4" />} onClick={() => setEditing("new")}>New category</Button>
      </div>

      {categories.isLoading ? (
        <div className="h-48 animate-pulse rounded-2xl bg-gray-100 dark:bg-white/5" />
      ) : categories.isError ? (
        <ErrorState onRetry={() => categories.refetch()} />
      ) : (categories.data ?? []).length === 0 ? (
        <EmptyState title="No categories yet" />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableCell isHeader>Code</TableCell>
              <TableCell isHeader>Name</TableCell>
              <TableCell isHeader>Monthly limit</TableCell>
              <TableCell isHeader>Status</TableCell>
              <TableCell isHeader className="!text-right">Actions</TableCell>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(categories.data ?? []).map((c) => (
              <TableRow key={c.id}>
                <TableCell className="font-mono text-xs">{c.code}</TableCell>
                <TableCell className="font-medium">{c.name}</TableCell>
                <TableCell>{c.monthly_limit_per_employee ? fmtIDR(c.monthly_limit_per_employee) : "—"}</TableCell>
                <TableCell><Badge color={c.is_active ? "success" : "light"}>{c.is_active ? "active" : "inactive"}</Badge></TableCell>
                <TableCell className="!text-right">
                  <Button variant="outline" size="xs" onClick={() => setEditing(c)}>Edit</Button>{" "}
                  <Button variant="ghost" size="xs" onClick={() => setDeleting(c)}>Delete</Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <CategoryModal editing={editing} onClose={() => setEditing(null)} />
      <ConfirmDialog
        isOpen={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={() => deleting && del.mutate(deleting.id, { onSuccess: () => setDeleting(null) })}
        title="Delete category?"
        message={deleting ? `"${deleting.name}" will be removed.` : ""}
        tone="danger"
        loading={del.isPending}
      />
    </div>
  );
}

function CategoryModal({ editing, onClose }: { editing: Category | "new" | null; onClose: () => void }) {
  const isNew = editing === "new";
  const cat = !isNew && editing !== null && typeof editing === "object" ? editing : null;
  const save = useSaveCategory(cat?.id);

  return (
    <Modal isOpen={editing !== null} onClose={onClose} title={isNew ? "New category" : "Edit category"} className="max-w-md">
      {editing !== null && (
        <CategoryForm
          key={cat?.id ?? "new"}
          initial={{
            code: cat?.code ?? "",
            name: cat?.name ?? "",
            limit: cat?.monthly_limit_per_employee ?? "",
            active: cat?.is_active ?? true,
            locked: !isNew,
          }}
          saving={save.isPending}
          onSave={(v) =>
            save.mutate(
              {
                code: v.code.trim(),
                name: v.name.trim(),
                monthly_limit_per_employee: v.limit.trim() || null,
                is_active: v.active,
              },
              { onSuccess: onClose },
            )
          }
          onCancel={onClose}
        />
      )}
    </Modal>
  );
}

function CategoryForm({
  initial,
  saving,
  onSave,
  onCancel,
}: {
  initial: { code: string; name: string; limit: string; active: boolean; locked: boolean };
  saving: boolean;
  onSave: (v: { code: string; name: string; limit: string; active: boolean }) => void;
  onCancel: () => void;
}) {
  const [code, setCode] = useState(initial.code);
  const [name, setName] = useState(initial.name);
  const [limit, setLimit] = useState(initial.limit);
  const [active, setActive] = useState(initial.active);

  return (
    <form
      className="space-y-4 p-5"
      onSubmit={(e) => {
        e.preventDefault();
        onSave({ code, name, limit, active });
      }}
    >
      <FormField label="Code" required hint="Short uppercase identifier, e.g. TRV">
        <Input value={code} onChange={(e) => setCode(e.target.value)} required disabled={initial.locked} placeholder="TRV" />
      </FormField>
      <FormField label="Name" required>
        <Input value={name} onChange={(e) => setName(e.target.value)} required placeholder="Travel & Transportation" />
      </FormField>
      <FormField label="Monthly limit per employee (Rp)" hint="Empty = no limit.">
        <Input type="number" min={0} step="any" value={limit} onChange={(e) => setLimit(e.target.value)} placeholder="5000000" />
      </FormField>
      <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
        <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} /> Active
      </label>
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" type="button" onClick={onCancel}>Cancel</Button>
        <Button size="sm" type="submit" loading={saving}>Save</Button>
      </div>
    </form>
  );
}

// --- Departments ---
export function AdminDepartmentsPage() {
  const departments = useDepartments();
  const [editing, setEditing] = useState<Department | "new" | null>(null);
  const [deleting, setDeleting] = useState<Department | null>(null);
  const del = useDeleteDepartment();
  const save = useSaveDepartment(editing && editing !== "new" ? editing.id : undefined);

  const [name, setName] = useState("");
  const [budget, setBudget] = useState("");

  function openEdit(d: Department | "new") {
    if (d === "new") { setName(""); setBudget(""); }
    else { setName(d.name); setBudget(d.monthly_budget ?? ""); }
    setEditing(d);
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="text-title-md font-semibold text-gray-800 dark:text-white/90">Departments</h1>
        <Button size="sm" startIcon={<Plus className="size-4" />} onClick={() => openEdit("new")}>New department</Button>
      </div>

      {departments.isLoading ? (
        <div className="h-40 animate-pulse rounded-2xl bg-gray-100 dark:bg-white/5" />
      ) : departments.isError ? (
        <ErrorState onRetry={() => departments.refetch()} />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableCell isHeader>Name</TableCell>
              <TableCell isHeader>Monthly budget</TableCell>
              <TableCell isHeader className="!text-right">Actions</TableCell>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(departments.data ?? []).map((d) => (
              <TableRow key={d.id}>
                <TableCell className="font-medium">{d.name}</TableCell>
                <TableCell>{d.monthly_budget ? fmtIDR(d.monthly_budget) : "—"}</TableCell>
                <TableCell className="!text-right">
                  <Button variant="outline" size="xs" onClick={() => openEdit(d)}>Edit</Button>{" "}
                  <Button variant="ghost" size="xs" onClick={() => setDeleting(d)}>Delete</Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <Modal isOpen={editing !== null} onClose={() => setEditing(null)} title={editing === "new" ? "New department" : "Edit department"}>
        <form
          className="space-y-4 p-5"
          onSubmit={(e) => {
            e.preventDefault();
            save.mutate(
              { name: name.trim(), monthly_budget: budget.trim() || null },
              { onSuccess: () => setEditing(null) },
            );
          }}
        >
          <FormField label="Name" required>
            <Input value={name} onChange={(e) => setName(e.target.value)} required placeholder="Engineering" />
          </FormField>
          <FormField label="Monthly budget (Rp)" hint="Shown as usage bar on dashboards. Empty = untracked.">
            <Input type="number" min={0} step="any" value={budget} onChange={(e) => setBudget(e.target.value)} placeholder="50000000" />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" type="button" onClick={() => setEditing(null)}>Cancel</Button>
            <Button size="sm" type="submit" loading={save.isPending}>Save</Button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        isOpen={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={() => deleting && del.mutate(deleting.id, { onSuccess: () => setDeleting(null) })}
        title="Delete department?"
        message={deleting ? `"${deleting.name}" will be removed. Users in it must be moved first.` : ""}
        tone="danger"
        loading={del.isPending}
      />
    </div>
  );
}

// --- Users ---
const ROLES = ["employee", "manager", "finance", "admin"];

export function AdminUsersPage() {
  const [page] = useState(1);
  const [search, setSearch] = useState("");
  const users = useUsers(page, search);
  const create = useCreateUser();
  const resetPw = useResetPassword();

  const [creating, setCreating] = useState(false);
  const [resetting, setResetting] = useState<UserRow | null>(null);
  const [deactivating, setDeactivating] = useState<UserRow | null>(null);

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState("employee");
  const [deptId, setDeptId] = useState("");
  const [newPw, setNewPw] = useState("");
  const departments = useDepartments();

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-title-md font-semibold text-gray-800 dark:text-white/90">Users</h1>
        <div className="flex gap-2">
          <form onSubmit={(e) => e.preventDefault()}>
            <Input placeholder="Search name / email…" value={search} onChange={(e) => setSearch(e.target.value)} className="!h-10 w-56" />
          </form>
          <Button size="sm" startIcon={<Plus className="size-4" />} onClick={() => setCreating(true)}>New user</Button>
        </div>
      </div>

      {users.isLoading ? (
        <div className="h-64 animate-pulse rounded-2xl bg-gray-100 dark:bg-white/5" />
      ) : users.isError ? (
        <ErrorState onRetry={() => users.refetch()} />
      ) : (users.data?.items.length ?? 0) === 0 ? (
        <EmptyState title="No users found" />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableCell isHeader>Name</TableCell>
              <TableCell isHeader>Email</TableCell>
              <TableCell isHeader>Role</TableCell>
              <TableCell isHeader>Department</TableCell>
              <TableCell isHeader>Status</TableCell>
              <TableCell isHeader className="!text-right">Actions</TableCell>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.data!.items.map((u) => (
              <TableRow key={u.id}>
                <TableCell className="font-medium">{u.name}</TableCell>
                <TableCell>{u.email}</TableCell>
                <TableCell><Badge color={u.role === "admin" ? "dark" : u.role === "finance" ? "info" : "primary"}>{u.role}</Badge></TableCell>
                <TableCell>{u.department_name ?? "—"}</TableCell>
                <TableCell><Badge color={u.is_active === false ? "error" : "success"}>{u.is_active === false ? "inactive" : "active"}</Badge></TableCell>
                <TableCell className="!text-right">
                  <div className="flex justify-end gap-1.5">
                    <Button variant="ghost" size="xs" startIcon={<KeyRound className="size-3.5" />} onClick={() => { setResetting(u); setNewPw(""); }}>
                      Reset PW
                    </Button>
                    <Button
                      variant="ghost"
                      size="xs"
                      onClick={() => setDeactivating(u)}
                    >
                      {u.is_active === false ? "Activate" : "Deactivate"}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* Create modal */}
      <Modal isOpen={creating} onClose={() => setCreating(false)} title="Create user">
        <form
          className="space-y-4 p-5"
          onSubmit={(e) => {
            e.preventDefault();
            create.mutate(
              { name: name.trim(), email: email.trim(), password, role, department_id: deptId || null },
              { onSuccess: () => { setCreating(false); setPassword(""); } },
            );
          }}
        >
          <FormField label="Name" required><Input value={name} onChange={(e) => setName(e.target.value)} required /></FormField>
          <FormField label="Email" required>
            <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </FormField>
          <FormField label="Temporary password" required hint="Min 8 characters. User should change it later.">
            <Input type="password" minLength={8} value={password} onChange={(e) => setPassword(e.target.value)} required />
          </FormField>
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Role" required>
              <Select value={role} onChange={(e) => setRole(e.target.value)} options={ROLES.map((r) => ({ value: r, label: r }))} />
            </FormField>
            <FormField label="Department">
              <Select
                value={deptId}
                onChange={(e) => setDeptId(e.target.value)}
                options={[
                  { value: "", label: "—" },
                  ...(departments.data ?? []).map((d) => ({ value: d.id, label: d.name })),
                ]}
              />
            </FormField>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" type="button" onClick={() => setCreating(false)}>Cancel</Button>
            <Button size="sm" type="submit" loading={create.isPending}>Create</Button>
          </div>
        </form>
      </Modal>

      {/* Reset password modal */}
      <Modal isOpen={resetting !== null} onClose={() => setResetting(null)} title={`Reset password — ${resetting?.name ?? ""}`}>
        <form
          className="space-y-4 p-5"
          onSubmit={(e) => {
            e.preventDefault();
            resetting &&
              resetPw.mutate(
                { id: resetting.id, new_password: newPw },
                { onSuccess: () => setResetting(null) },
              );
          }}
        >
          <FormField label="New password" required hint="Min 8 characters.">
            <Input type="password" minLength={8} value={newPw} onChange={(e) => setNewPw(e.target.value)} required autoFocus />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" type="button" onClick={() => setResetting(null)}>Cancel</Button>
            <Button size="sm" type="submit" loading={resetPw.isPending}>Reset</Button>
          </div>
        </form>
      </Modal>

      {/* Deactivate confirm */}
      <ConfirmDialog
        isOpen={deactivating !== null}
        onClose={() => setDeactivating(null)}
        onConfirm={() =>
          deactivating &&
          api
            .patch(`/users/${deactivating.id}`, { is_active: deactivating.is_active === false })
            .then(() => {
              toast.success(deactivating.is_active === false ? "User reactivated" : "User deactivated");
              users.refetch();
              setDeactivating(null);
            })
            .catch((e) => toast.error(e?.response?.data?.error?.message ?? "Update failed"))
        }
        title={deactivating?.is_active === false ? "Reactivate user?" : "Deactivate user?"}
        message={
          deactivating
            ? deactivating.is_active === false
              ? `${deactivating.name} will be able to log in again.`
              : `${deactivating.name} will be blocked from logging in.`
            : ""
        }
        tone={deactivating?.is_active === false ? "primary" : "warning"}
        loading={false}
      />
    </div>
  );
}
