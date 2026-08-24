import { Routes, Route, Navigate } from "react-router-dom";
import AppLayout from "./components/layout/AppLayout";
import ProtectedRoute from "./components/routing/ProtectedRoute";
import RoleRoute from "./components/routing/RoleRoute";
import LoginPage from "./pages/LoginPage";
import DashboardPage from "./pages/DashboardPage";
import ClaimsListPage from "./pages/ClaimsListPage";
import ClaimDetailPage from "./pages/ClaimDetailPage";
import ClaimFormPage from "./pages/ClaimFormPage";
import ClaimsQueuePage from "./pages/ClaimsQueuePage";
import { AdminUsersPage, AdminCategoriesPage } from "./pages/AdminPages";
import { ForbiddenPage, NotFoundPage } from "./pages/ErrorPages";

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<ProtectedRoute />}>
        <Route element={<AppLayout />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="reimbursements" element={<ClaimsListPage />} />
          <Route
            path="reimbursements/new"
            element={
              <RoleRoute allow={["employee", "manager", "finance", "admin"]}>
                <ClaimFormPage mode="create" />
              </RoleRoute>
            }
          />
          <Route path="reimbursements/:id" element={<ClaimDetailPage />} />
          <Route
            path="reimbursements/:id/edit"
            element={
              <RoleRoute allow={["employee", "manager", "finance", "admin"]}>
                <ClaimFormPage mode="edit" />
              </RoleRoute>
            }
          />
          <Route
            path="approvals"
            element={
              <RoleRoute allow={["manager", "finance", "admin"]}>
                <ClaimsQueuePage mode="approvals" />
              </RoleRoute>
            }
          />
          <Route
            path="payments"
            element={
              <RoleRoute allow={["finance"]}>
                <ClaimsQueuePage mode="payments" />
              </RoleRoute>
            }
          />
          <Route
            path="admin/users"
            element={<RoleRoute allow={["admin"]}><AdminUsersPage /></RoleRoute>}
          />
          <Route
            path="admin/categories"
            element={<RoleRoute allow={["admin"]}><AdminCategoriesPage /></RoleRoute>}
          />
          <Route path="403" element={<ForbiddenPage />} />
          <Route path="404" element={<NotFoundPage />} />
          <Route path="*" element={<Navigate to="/404" replace />} />
        </Route>
      </Route>
    </Routes>
  );
}
