import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { isAuthenticationFailure, setAccessToken as setClientAccessToken, setAuthStateListener } from "../api/client";
import { getCSRFCookie } from "../api/csrf";
import { getCurrentUser, login as loginApi, logout as logoutApi, refresh as refreshApi, signUp as signUpApi, withdraw as withdrawApi } from "../api/user";
import type { AuthUser, LoginInput, SignUpInput, WithdrawInput } from "../types/user";
import { isUserRole } from "../constants/role";
import { clearQuestionAnswers } from "../utils/questionAnswers";

interface AuthContextValue { user: AuthUser | null; accessToken: string | null; isAuthenticated: boolean; isLoading: boolean; authExpired: boolean; signUp: (input: SignUpInput) => Promise<void>; login: (input: LoginInput) => Promise<AuthUser>; logout: () => Promise<void>; withdraw: (input: WithdrawInput) => Promise<string>; clearAuth: () => void; }
const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [authExpired, setAuthExpired] = useState(false);

  const clearAuth = (): void => { setAccessToken(null); setClientAccessToken(null); setUser(null); };

  useEffect(() => {
    setAuthStateListener((token, reason) => {
      setAccessToken(token);
      if (token === null) setUser(null);
      if (reason === "expired") setAuthExpired(true);
    });
    let active = true;
    const restore = async (): Promise<void> => {
      try {
        if (!getCSRFCookie()) {
          if (active) clearAuth();
          return;
        }
        const refreshed = await refreshApi();
        if (!active) return;
        setAccessToken(refreshed.access_token);
        setClientAccessToken(refreshed.access_token);
        const currentUser = await getCurrentUser();
        if (!isUserRole(currentUser.role)) throw new Error("invalid role");
        if (active) setUser(currentUser);
      } catch (error: unknown) {
        if (active) {
          clearAuth();
          if (isAuthenticationFailure(error)) setAuthExpired(true);
        }
      } finally {
        if (active) setIsLoading(false);
      }
    };
    void restore();
    return () => { active = false; setAuthStateListener(null); };
  }, []);

  const signUp = async (input: SignUpInput): Promise<void> => {
    setIsLoading(true);
    try {
      const response = await signUpApi(input);
      setAuthExpired(false);
      setAccessToken(response.access_token);
      setClientAccessToken(response.access_token);
      if (!isUserRole(response.user.role)) throw new Error("invalid role");
      setUser(response.user);
    } catch (error: unknown) {
      clearAuth();
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  const login = async (input: LoginInput): Promise<AuthUser> => {
    setIsLoading(true);
    try {
      const response = await loginApi(input);
      setAuthExpired(false);
      setAccessToken(response.access_token);
      setClientAccessToken(response.access_token);
      const currentUser = await getCurrentUser();
      if (!isUserRole(currentUser.role)) throw new Error("invalid role");
      setUser(currentUser);
      return currentUser;
    } catch (error: unknown) {
      clearAuth();
      throw error;
    } finally { setIsLoading(false); }
  };

  const withdraw = async (input: WithdrawInput): Promise<string> => {
    const response = await withdrawApi(input);
    clearAuth();
    setAuthExpired(false);
    return response.message;
  };

  const logout = async (): Promise<void> => {
    const currentUserId = user?.id;
    try {
      await logoutApi();
    } finally {
      if (currentUserId) clearQuestionAnswers(currentUserId);
      clearAuth();
    }
  };

  const value = useMemo(() => ({ user, accessToken, isAuthenticated: user !== null && accessToken !== null, isLoading, authExpired, signUp, login, logout, withdraw, clearAuth }), [user, accessToken, isLoading, authExpired]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within an AuthProvider");
  return context;
}
