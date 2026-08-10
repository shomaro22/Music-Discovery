import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { bootstrapCSRF, isAuthenticationFailure, setAccessToken as setClientAccessToken, setAuthStateListener } from "../api/client";
import { clearCSRFToken } from "../api/csrf";
import { getCurrentUser, login as loginApi, logout as logoutApi, refresh as refreshApi, signUp as signUpApi, withdraw as withdrawApi, updateProfile as updateProfileApi } from "../api/user";
import type { AuthUser, LoginInput, SignUpInput, WithdrawInput, UpdateProfileInput } from "../types/user";
import { isUserRole } from "../constants/role";
import { clearQuestionAnswers } from "../utils/questionAnswers";

interface AuthContextValue { user: AuthUser | null; accessToken: string | null; isAuthenticated: boolean; isLoading: boolean; authExpired: boolean; signUp: (input: SignUpInput) => Promise<void>; login: (input: LoginInput) => Promise<AuthUser>; logout: () => Promise<void>; withdraw: (input: WithdrawInput) => Promise<string>; updateProfile: (input: UpdateProfileInput) => Promise<void>; clearAuth: () => void; }
const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [authExpired, setAuthExpired] = useState(false);
  const currentUserIDRef = useRef<string | null>(null);

  const updateUser = useCallback((nextUser: AuthUser | null): void => {
    currentUserIDRef.current = nextUser?.id ?? null;
    setUser(nextUser);
  }, []);

  const clearLocalAuth = useCallback((): void => {
    const currentUserID = currentUserIDRef.current;
    if (currentUserID) clearQuestionAnswers(currentUserID);
    currentUserIDRef.current = null;
    setAccessToken(null);
    setUser(null);
  }, []);

  const clearAuth = useCallback((): void => {
    clearLocalAuth();
    setClientAccessToken(null);
    clearCSRFToken();
  }, [clearLocalAuth]);

  useEffect(() => {
    setAuthStateListener((token, reason) => {
      setAccessToken(token);
      if (token === null) clearLocalAuth();
      if (reason === "expired") setAuthExpired(true);
    });
    let active = true;
    const restore = async (): Promise<void> => {
      try {
        const csrf = await bootstrapCSRF();
        if (!csrf.refresh_available) {
          if (active) clearAuth();
          return;
        }
        const refreshed = await refreshApi();
        if (!active) return;
        setAccessToken(refreshed.access_token);
        setClientAccessToken(refreshed.access_token);
        const currentUser = await getCurrentUser();
        if (!isUserRole(currentUser.role)) throw new Error("invalid role");
        if (active) updateUser(currentUser);
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
  }, [clearAuth, clearLocalAuth, updateUser]);

  const signUp = async (input: SignUpInput): Promise<void> => {
    setIsLoading(true);
    try {
      const response = await signUpApi(input);
      setAuthExpired(false);
      setAccessToken(response.access_token);
      setClientAccessToken(response.access_token);
      if (!isUserRole(response.user.role)) throw new Error("invalid role");
      updateUser(response.user);
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
      updateUser(currentUser);
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


  const updateProfile = async (input: UpdateProfileInput): Promise<void> => {
    const updated = await updateProfileApi(input);
    updateUser(updated);
  };

  const logout = async (): Promise<void> => {
    try {
      await logoutApi();
    } finally {
      clearAuth();
    }
  };

  const value = useMemo(() => ({ user, accessToken, isAuthenticated: user !== null && accessToken !== null, isLoading, authExpired, signUp, login, logout, withdraw, updateProfile, clearAuth }), [user, accessToken, isLoading, authExpired, clearAuth]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within an AuthProvider");
  return context;
}
