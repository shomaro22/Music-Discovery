import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "../auth/AuthContext";
import { ROLE } from "../constants/role";
import { AdminPlaceholderPage } from "../pages/AdminPlaceholderPage";
import { ForbiddenPage } from "../pages/ForbiddenPage";
import { LoginPage } from "../pages/LoginPage";
import { SignupPage } from "../pages/SignupPage";
import { TopPage } from "../pages/TopPage";
import { WithdrawPage } from "../pages/WithdrawPage";
import { AdminRoute } from "./AdminRoute";
import { ProtectedRoute } from "./ProtectedRoute";

export function AppRouter() {
  return <BrowserRouter><AuthProvider><Routes>
    <Route path="/signup" element={<SignupPage />} />
    <Route path="/login" element={<LoginPage />} />
    <Route path="/forbidden" element={<ProtectedRoute><ForbiddenPage /></ProtectedRoute>} />
    <Route path="/withdraw" element={<AdminRoute allowedRoles={[ROLE.USER]}><WithdrawPage /></AdminRoute>} />
    <Route path="/admin/content" element={<AdminRoute allowedRoles={[ROLE.CONTENT_ADMIN]}><AdminPlaceholderPage title="Content管理" /></AdminRoute>} />
    <Route path="/system/users" element={<AdminRoute allowedRoles={[ROLE.SYSTEM_ADMIN]}><AdminPlaceholderPage title="System管理" /></AdminRoute>} />
    <Route path="/" element={<ProtectedRoute><TopPage /></ProtectedRoute>} />
    <Route path="*" element={<Navigate to="/login" replace />} />
  </Routes></AuthProvider></BrowserRouter>;
}
