import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "../auth/AuthContext";
import { ROLE } from "../constants/role";
import { ROUTES } from "../constants/routes";
import { ContentAdminPage } from "../pages/admin/ContentAdminPage";
import { SystemAdminPage } from "../pages/admin/SystemAdminPage";
import { ForbiddenPage } from "../pages/ForbiddenPage";
import { LoginPage } from "../pages/LoginPage";
import { SignupPage } from "../pages/SignupPage";
import { TopPage } from "../pages/TopPage";
import { WithdrawPage } from "../pages/WithdrawPage";
import { QuestionPage } from "../pages/QuestionPage";
import { AnswerConfirmPage } from "../pages/AnswerConfirmPage";
import { RecommendationPage } from "../pages/RecommendationPage";
import { AdminRoute } from "./AdminRoute";
import { ProtectedRoute } from "./ProtectedRoute";

export function AppRouter() {
  return <BrowserRouter><AuthProvider><Routes>
    <Route path={ROUTES.signup} element={<SignupPage />} />
    <Route path={ROUTES.login} element={<LoginPage />} />
    <Route path={ROUTES.forbidden} element={<ProtectedRoute><ForbiddenPage /></ProtectedRoute>} />
    <Route path={ROUTES.withdraw} element={<AdminRoute allowedRoles={[ROLE.USER]}><WithdrawPage /></AdminRoute>} />
    <Route path={ROUTES.questions} element={<AdminRoute allowedRoles={[ROLE.USER, ROLE.CONTENT_ADMIN, ROLE.SYSTEM_ADMIN]}><QuestionPage /></AdminRoute>} />
    <Route path={ROUTES.questionConfirm} element={<AdminRoute allowedRoles={[ROLE.USER, ROLE.CONTENT_ADMIN, ROLE.SYSTEM_ADMIN]}><AnswerConfirmPage /></AdminRoute>} />
    <Route path={ROUTES.recommendations} element={<AdminRoute allowedRoles={[ROLE.USER, ROLE.CONTENT_ADMIN, ROLE.SYSTEM_ADMIN]}><RecommendationPage /></AdminRoute>} />
    <Route path={ROUTES.contentAdmin} element={<AdminRoute allowedRoles={[ROLE.CONTENT_ADMIN]}><ContentAdminPage /></AdminRoute>} />
    <Route path={ROUTES.systemUsers} element={<AdminRoute allowedRoles={[ROLE.SYSTEM_ADMIN]}><SystemAdminPage /></AdminRoute>} />
    <Route path={ROUTES.top} element={<ProtectedRoute><TopPage /></ProtectedRoute>} />
    <Route path="*" element={<Navigate to={ROUTES.login} replace />} />
  </Routes></AuthProvider></BrowserRouter>;
}
