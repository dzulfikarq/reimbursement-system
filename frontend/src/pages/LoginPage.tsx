import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useNavigate } from "react-router-dom";
import { AxiosError } from "axios";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import FormField from "../components/ui/FormField";
import { useAuthStore } from "../stores/auth";
import type { ApiErrorBody } from "../lib/api";
import { ReceiptText, AlertCircle } from "lucide-react";

const loginSchema = z.object({
  email: z.string().min(1, "Email is required").email("Invalid email address"),
  password: z.string().min(8, "Password must be at least 8 characters"),
});

type LoginForm = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const navigate = useNavigate();
  const login = useAuthStore((s) => s.login);
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });

  const onSubmit = async (values: LoginForm) => {
    try {
      await login(values.email, values.password);
      navigate("/", { replace: true });
    } catch (err) {
      if (err instanceof AxiosError && err.response?.data) {
        const body = err.response.data as ApiErrorBody;
        // Map backend 422 details onto fields; otherwise surface message.
        if (body.details?.length) {
          for (const d of body.details) {
            const field = d.field.toLowerCase() as keyof LoginForm;
            if (field in values) setError(field, { message: d.message });
          }
        } else {
          setError("root", { message: body.message ?? "Login failed" });
        }
      } else {
        setError("root", { message: "Network error — please try again" });
      }
    }
  };

  return (
    <main className="flex min-h-screen items-center justify-center bg-gray-50 p-4 dark:bg-gray-900">
      <div className="w-full max-w-md">
        <div className="mb-8 flex flex-col items-center gap-3">
          <div className="flex size-12 items-center justify-center rounded-xl bg-brand-500 text-white shadow-theme-md">
            <ReceiptText className="size-6" />
          </div>
          <h1 className="text-2xl font-semibold text-gray-800 dark:text-white/90">
            Reimburse<span className="text-brand-500">Flow</span>
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Sign in to your account
          </p>
        </div>

        <form
          onSubmit={handleSubmit(onSubmit)}
          noValidate
          className="rounded-2xl border border-gray-200 bg-white p-6 shadow-theme-sm dark:border-gray-800 dark:bg-white/[0.03] sm:p-8"
        >
          <div className="space-y-5">
            {errors.root?.message && (
              <div className="flex items-start gap-2 rounded-lg bg-error-50 p-3 text-sm text-error-600 dark:bg-error-500/10 dark:text-error-400">
                <AlertCircle className="mt-0.5 size-4 shrink-0" />
                {errors.root.message}
              </div>
            )}

            <FormField label="Email" htmlFor="email" required error={errors.email?.message}>
              <Input
                id="email"
                type="email"
                autoComplete="email"
                placeholder="you@company.com"
                {...register("email")}
              />
            </FormField>

            <FormField label="Password" htmlFor="password" required error={errors.password?.message}>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                placeholder="••••••••"
                {...register("password")}
              />
            </FormField>

            <Button type="submit" className="w-full" disabled={isSubmitting}>
              {isSubmitting ? "Signing in…" : "Sign In"}
            </Button>
          </div>
        </form>
      </div>
    </main>
  );
}
