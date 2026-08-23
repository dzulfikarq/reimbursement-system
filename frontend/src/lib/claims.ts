import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import { api } from "./api";
import { toast, errMessage } from "../stores/toast";

// --- shared shapes (mirror backend dto.go / docs/04) ---

export interface ClaimItem {
  id: string;
  description: string;
  quantity: number;
  unit_price: string;
  line_total: string;
}

export interface Attachment {
  id: string;
  original_filename: string;
  mime_type: string;
  size_bytes: number;
  created_at: string;
}

export interface ApprovalStep {
  step_number: number;
  approver_role: string;
  status: string;
  note?: string;
}

export interface Claim {
  id: string;
  employee_id: string;
  employee_name?: string;
  category_id: string;
  category_name?: string;
  category_code?: string;
  title: string;
  description: string;
  expense_date: string;
  amount: string;
  status: string;
  current_step?: number;
  created_at: string;
  updated_at: string;
}

export interface ClaimDetail extends Claim {
  items: ClaimItem[];
  attachments: Attachment[];
  approvals: ApprovalStep[];
}

export interface ListMeta {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface ListParams {
  page?: number;
  limit?: number;
  search?: string;
  sort?: string;
  order?: "asc" | "desc";
  status?: string;
  category_id?: string;
  date_from?: string;
  date_to?: string;
}

function toQueryString(p: ListParams): string {
  const qs = new URLSearchParams();
  Object.entries(p).forEach(([k, v]) => {
    if (v !== undefined && v !== "" && v !== null) qs.set(k, String(v));
  });
  return qs.toString();
}

// --- claims ---

export interface FlatList {
  items: Claim[];
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export type ClaimsPageData = { items: Claim[]; meta: ListMeta };

// Backend list shape is flat: {items, page, limit, total, total_pages}.
export function useClaims(params: ListParams) {
  return useQuery({
    queryKey: ["claims", params],
    queryFn: async (): Promise<ClaimsPageData> => {
      const res = await api.get<{ data: FlatList }>(`/reimbursements?${toQueryString(params)}`);
      const d = res.data.data;
      return {
        items: d.items ?? [],
        meta: { page: d.page, limit: d.limit, total: d.total, total_pages: d.total_pages },
      };
    },
    placeholderData: keepPreviousData,
  });
}

export function useClaim(id: string | undefined) {
  return useQuery({
    queryKey: ["claim", id],
    enabled: !!id,
    queryFn: async () => {
      const res = await api.get<{ data: ClaimDetail }>(`/reimbursements/${id}`);
      return res.data.data;
    },
  });
}

export interface ClaimFormInput {
  category_id: string;
  title: string;
  description?: string;
  expense_date: string;
  items: { description: string; quantity: number; unit_price: string }[];
}

export function useCreateClaim() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ClaimFormInput) => {
      const res = await api.post<{ data: ClaimDetail }>("/reimbursements", input);
      return res.data.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["claims"] });
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

export function useUpdateClaim(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ClaimFormInput) => {
      const res = await api.patch<{ data: ClaimDetail }>(`/reimbursements/${id}`, input);
      return res.data.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["claims"] });
      qc.invalidateQueries({ queryKey: ["claim", id] });
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

export type WorkflowAction = "submit" | "approve" | "reject" | "cancel" | "pay";

export function useClaimAction(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ action, note }: { action: WorkflowAction; note?: string }) => {
      const res = await api.post<{ data: ClaimDetail; message?: string }>(
        `/reimbursements/${id}/${action}`,
        note ? { note } : {},
      );
      return res.data;
    },
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ["claims"] });
      qc.invalidateQueries({ queryKey: ["claim", id] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
      if (res.message) toast.success(res.message);
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

export function useDeleteClaim(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => api.delete(`/reimbursements/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["claims"] });
      toast.success("Claim deleted");
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

export function useUploadAttachment(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (file: File) => {
      const form = new FormData();
      form.append("file", file);
      const res = await api.post(`/reimbursements/${id}/attachments`, form, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      return res.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["claim", id] });
      toast.success("Receipt uploaded");
    },
    onError: (e) => toast.error(errMessage(e)),
  });
}

// --- dashboard ---

export interface DepartmentUsage {
  department_id: string;
  department_name: string;
  monthly_budget: string;
  monthly_spend: string;
  used_percent: number;
}

export interface DashboardSummary {
  pending_count: number;
  monthly_total: string;
  approval_rate: number | null;
  budget_usage: DepartmentUsage[];
}

export function useDashboardSummary() {
  return useQuery({
    queryKey: ["dashboard", "summary"],
    queryFn: async () => {
      const res = await api.get<{ data: DashboardSummary }>("/dashboard/summary");
      return res.data.data;
    },
    refetchInterval: 60_000,
  });
}

export interface TrendPoint {
  month: string;
  total: string;
}

export function useMonthlyTrend(months = 6) {
  return useQuery({
    queryKey: ["dashboard", "trend", months],
    queryFn: async () => {
      const res = await api.get<{ data: { series: TrendPoint[] } }>(`/dashboard/monthly-trend?months=${months}`);
      return res.data.data.series;
    },
  });
}

export interface CategoryBreakdownItem {
  category_id: string;
  category_name: string;
  total: string;
  claim_count: number;
}

export function useCategoryBreakdown(month?: string) {
  return useQuery({
    queryKey: ["dashboard", "breakdown", month],
    queryFn: async () => {
      const q = month ? `?month=${month}` : "";
      const res = await api.get<{ data: { items: CategoryBreakdownItem[] } }>(`/dashboard/category-breakdown${q}`);
      return res.data.data.items;
    },
  });
}
