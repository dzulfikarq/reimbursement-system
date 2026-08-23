import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "./api";
import { toast, errMessage } from "../stores/toast";

export interface Category {
  id: string;
  code: string;
  name: string;
  monthly_limit_per_employee: string | null;
  is_active: boolean;
}

export interface Department {
  id: string;
  name: string;
  monthly_budget: string | null;
}

export interface UserRow {
  id: string;
  name: string;
  email: string;
  role: "employee" | "manager" | "finance" | "admin";
  department_id: string | null;
  department_name?: string | null;
  is_active?: boolean;
}

function unwrapList<T>(res: { data: { items?: T[] } | T[] }): T[] {
  const d = res.data;
  return Array.isArray(d) ? d : d.items ?? [];
}

// --- categories ---

export function useCategories() {
  return useQuery({
    queryKey: ["categories"],
    queryFn: async () => unwrapList<Category>(await api.get("/categories?limit=100")),
  });
}

export function useSaveCategory(id?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: Partial<Category>) =>
      id ? api.patch(`/categories/${id}`, input) : api.post("/categories", input),
    onSuccess: (_r, _v) => {
      qc.invalidateQueries({ queryKey: ["categories"] });
      toast.success(id ? "Category updated" : "Category created");
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

export function useDeleteCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/categories/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["categories"] });
      toast.success("Category deleted");
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

// --- departments ---

export function useDepartments() {
  return useQuery({
    queryKey: ["departments"],
    queryFn: async () => unwrapList<Department>(await api.get("/departments?limit=100")),
  });
}

export function useSaveDepartment(id?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: Partial<Department>) =>
      id ? api.patch(`/departments/${id}`, input) : api.post("/departments", input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["departments"] });
      toast.success(id ? "Department updated" : "Department created");
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

export function useDeleteDepartment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/departments/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["departments"] });
      toast.success("Department deleted");
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

// --- users (admin only) ---

interface UsersPage {
  items: UserRow[];
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export function useUsers(page = 1, search = "") {
  return useQuery({
    queryKey: ["users", page, search],
    queryFn: async () => {
      const qs = new URLSearchParams({ page: String(page), limit: "10" });
      if (search) qs.set("search", search);
      const res = await api.get<{ data: UsersPage }>(`/admin/users?${qs.toString()}`);
      return res.data.data;
    },
  });
}

export interface UserInput {
  name: string;
  email: string;
  password: string;
  role: string;
  department_id: string | null;
}

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UserInput) => api.post("/users", input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      toast.success("User created");
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

export function useUpdateUser(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: Partial<UserInput> & { is_active?: boolean }) => {
      const res = await api.patch(`/users/${id}`, input);
      return res.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      toast.success("User updated");
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

export function useResetPassword() {
  return useMutation({
    mutationFn: async ({ id, new_password }: { id: string; new_password: string }) =>
      api.post(`/users/${id}/reset-password`, { new_password }),
    onSuccess: () => toast.success("Password reset"),
    onError: (e) => toast.error(errMessage(e)),
  });
}
