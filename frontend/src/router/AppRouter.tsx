import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "../auth/AuthContext";
import { ROLE } from "../constants/role";
import { ContentAdminPage } from "../pages/admin/ContentAdminPage";
import { SystemAdminPage } from "../pages/admin/SystemAdminPage";
import { ForbiddenPage } from "../pages/ForbiddenPage";
import { LoginPage } from "../pages/LoginPage";
import { SignupPage } from "../pages/SignupPage";
import { TopPage } from "../pages/TopPage";
import { WithdrawPage } from "../pages/WithdrawPage";
import { QuestionPage } from "../pages/QuestionPage";
import { AnswerConfirmPage } from "../pages/AnswerConfirmPage";
import { RecommendationPlaceholderPage } from "../pages/RecommendationPlaceholderPage";
import { AdminRoute } from "./AdminRoute";
import { ProtectedRoute } from "./ProtectedRoute";

export function AppRouter() {
  return <BrowserRouter><AuthProvider><Routes>
    <Route path="/signup" element={<SignupPage />} />
    <Route path="/login" element={<LoginPage />} />
    <Route path="/forbidden" element={<ProtectedRoute><ForbiddenPage /></ProtectedRoute>} />
    <Route path="/withdraw" element={<AdminRoute allowedRoles={[ROLE.USER]}><WithdrawPage /></AdminRoute>} />
    <Route path="/questions" element={<AdminRoute allowedRoles={[ROLE.USER]}><QuestionPage /></AdminRoute>} />
    <Route path="/questions/confirm" element={<AdminRoute allowedRoles={[ROLE.USER]}><AnswerConfirmPage /></AdminRoute>} />
    <Route path="/recommendations" element={<AdminRoute allowedRoles={[ROLE.USER]}><RecommendationPlaceholderPage /></AdminRoute>} />
    <Route path="/admin/content" element={<AdminRoute allowedRoles={[ROLE.CONTENT_ADMIN]}><ContentAdminPage /></AdminRoute>} />
    <Route path="/system/users" element={<AdminRoute allowedRoles={[ROLE.SYSTEM_ADMIN]}><SystemAdminPage /></AdminRoute>} />
    <Route path="/" element={<ProtectedRoute><TopPage /></ProtectedRoute>} />
    <Route path="*" element={<Navigate to="/login" replace />} />
  </Routes></AuthProvider></BrowserRouter>;
}
