import { useEffect, useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useFieldArray, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus, Trash2 } from "lucide-react";
import Card from "../components/ui/Card";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import Select from "../components/ui/Select";
import FormField from "../components/ui/FormField";
import { toast } from "../stores/toast";
import { fmtIDR } from "./ClaimsListPage";
import { useCategories } from "../lib/admin";
import { useCreateClaim, useUpdateClaim, useClaim } from "../lib/claims";

const itemSchema = z.object({
  description: z.string().min(1, "Required").max(200),
  quantity: z
    .string()
    .min(1, "Required")
    .refine((v) => /^\d+$/.test(v) && parseInt(v, 10) >= 1 && parseInt(v, 10) <= 10000, "1–10000"),
  unit_price: z
    .string()
    .min(1, "Required")
    .refine((v) => /^\d+(\.\d{1,2})?$/.test(v) && parseFloat(v) > 0, "Positive amount"),
});

const formSchema = z.object({
  category_id: z.string().uuid("Pick a category"),
  title: z.string().min(1, "Required").max(150),
  description: z.string().max(2000).optional(),
  expense_date: z.string().min(1, "Required").regex(/^\d{4}-\d{2}-\d{2}$/, "YYYY-MM-DD"),
  items: z.array(itemSchema).min(1, "Add at least one item").max(50),
});

type FormValues = z.infer<typeof formSchema>;

export default function ClaimFormPage({ mode }: { mode: "create" | "edit" }) {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const categories = useCategories();
  const existing = useClaim(mode === "edit" ? id : undefined);
  const create = useCreateClaim();
  const update = useUpdateClaim(id ?? "");

  const {
    register,
    control,
    handleSubmit,
    watch,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      category_id: "",
      title: "",
      description: "",
      expense_date: new Date().toISOString().slice(0, 10),
      items: [{ description: "", quantity: "1", unit_price: "" }],
    },
  });
  const { fields, append, remove } = useFieldArray({ control, name: "items" });

  useEffect(() => {
    if (mode === "edit" && existing.data) {
      const c = existing.data;
      reset({
        category_id: c.category_id,
        title: c.title,
        description: c.description ?? "",
        expense_date: c.expense_date,
        items: c.items.map((it) => ({
          description: it.description,
          quantity: String(it.quantity),
          unit_price: it.unit_price,
        })),
      });
    }
  }, [mode, existing.data, reset]);

  const items = watch("items");
  const total = useMemo(
    () => items.reduce((sum, it) => sum + (parseFloat(it.unit_price) || 0) * (Number(it.quantity) || 0), 0),
    [items],
  );
  const needsReceipt = total > 500_000;

  function onSubmit(values: FormValues) {
    const payload = {
      ...values,
      items: values.items.map((it) => ({
        description: it.description,
        quantity: parseInt(it.quantity, 10),
        unit_price: it.unit_price,
      })),
    };
    if (mode === "create") {
      create.mutate(payload, {
        onSuccess: (c) => {
          toast.success("Draft saved");
          navigate(`/reimbursements/${c.id}`);
        },
      });
    } else {
      update.mutate(payload, {
        onSuccess: () => {
          toast.success("Claim updated");
          navigate(`/reimbursements/${id}`);
        },
      });
    }
  }

  return (
    <div className="space-y-5">
      <h1 className="text-title-md font-semibold text-gray-800 dark:text-white/90">
        {mode === "create" ? "New reimbursement claim" : "Edit claim"}
      </h1>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" noValidate>
        <Card title="Details">
          <div className="grid grid-cols-1 gap-4 p-5 sm:grid-cols-2">
            <FormField label="Title" htmlFor="title" required error={errors.title?.message}>
              <Input id="title" placeholder="e.g. Client visit to Surabaya" {...register("title")} />
            </FormField>
            <FormField label="Category" htmlFor="category_id" required error={errors.category_id?.message}>
              <Select
                id="category_id"
                options={[
                  { value: "", label: "Select category…" },
                  ...(categories.data ?? [])
                    .filter((c) => c.is_active)
                    .map((c) => ({
                      value: c.id,
                      label:
                        c.monthly_limit_per_employee != null
                          ? `${c.name} (limit ${fmtIDR(c.monthly_limit_per_employee)}/mo)`
                          : c.name,
                    })),
                ]}
                {...register("category_id")}
              />
            </FormField>
            <FormField label="Expense date" htmlFor="expense_date" required error={errors.expense_date?.message}>
              <Input id="expense_date" type="date" {...register("expense_date")} />
            </FormField>
            <div className="sm:col-span-2">
              <FormField label="Description" htmlFor="description" error={errors.description?.message} hint="Optional context for approvers.">
                <textarea
                  id="description"
                  rows={3}
                  className="w-full rounded-lg border border-gray-300 bg-white px-4 py-3 text-sm text-gray-700 focus:border-brand-400 focus:outline-none dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300"
                  placeholder="Trip purpose, attendees…"
                  {...register("description")}
                />
              </FormField>
            </div>
          </div>
        </Card>

        <Card
          title="Items"
          desc={`Total ${fmtIDR(total)}${needsReceipt ? " · receipt required above Rp 500.000" : ""}`}
        >
          <div className="space-y-4 p-5">
            {typeof errors.items?.message === "string" && (
              <p className="text-xs text-error-500">{errors.items.message}</p>
            )}
            {fields.map((field, i) => (
              <div key={field.id} className="grid grid-cols-12 items-start gap-3">
                <div className="col-span-12 sm:col-span-6">
                  {i === 0 ? (
                    <FormField label="Description" error={errors.items?.[i]?.description?.message}>
                      <Input placeholder="e.g. Train ticket PP" {...register(`items.${i}.description`)} />
                    </FormField>
                  ) : (
                    <Input placeholder="e.g. Train ticket PP" {...register(`items.${i}.description`)} />
                  )}
                </div>
                <div className="col-span-4 sm:col-span-2">
                  {i === 0 ? (
                    <FormField label="Qty" error={errors.items?.[i]?.quantity?.message}>
                      <Input type="number" min={1} {...register(`items.${i}.quantity`)} />
                    </FormField>
                  ) : (
                    <Input type="number" min={1} {...register(`items.${i}.quantity`)} />
                  )}
                </div>
                <div className="col-span-6 sm:col-span-3">
                  {i === 0 ? (
                    <FormField label="Unit price" error={errors.items?.[i]?.unit_price?.message}>
                      <Input type="number" min={0} step="any" placeholder="0" {...register(`items.${i}.unit_price`)} />
                    </FormField>
                  ) : (
                    <Input type="number" min={0} step="any" placeholder="0" {...register(`items.${i}.unit_price`)} />
                  )}
                </div>
                <div className={`col-span-2 sm:col-span-1 ${i === 0 ? "sm:pt-8" : ""}`}>
                  <Button
                    variant="ghost"
                    size="xs"
                    disabled={fields.length <= 1 || create.isPending || update.isPending}
                    onClick={() => remove(i)}
                    aria-label="Remove item"
                  >
                    <Trash2 className="size-4 text-error-500" />
                  </Button>
                </div>
              </div>
            ))}
            <Button
              variant="outline"
              size="xs"
              startIcon={<Plus className="size-4" />}
              onClick={() => append({ description: "", quantity: "1", unit_price: "" })}
            >
              Add item
            </Button>
          </div>
        </Card>

        <div className="flex justify-end gap-3">
          <Button variant="outline" type="button" disabled={create.isPending || update.isPending} onClick={() => navigate(-1)}>
            Cancel
          </Button>
          <Button type="submit" loading={create.isPending || update.isPending}>
            {mode === "create" ? "Save draft" : "Save changes"}
          </Button>
        </div>
        {/* ponytail: receipt upload happens on the detail page after save */}
      </form>
    </div>
  );
}
