import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "../auth/AuthContext";
import { LoginPage } from "../pages/LoginPage";
import { SignupPage } from "../pages/SignupPage";
import { TopPage } from "../pages/TopPage";

export function AppRouter() {
  return <BrowserRouter><AuthProvider><Routes><Route path="/signup" element={<SignupPage />} /><Route path="/login" element={<LoginPage />} /><Route path="/" element={<TopPage />} /><Route path="*" element={<Navigate to="/login" replace />} /></Routes></AuthProvider></BrowserRouter>;
}
