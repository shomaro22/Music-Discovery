import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { setAccessToken as setClientAccessToken, setAuthStateListener } from "../api/client";
import { getCSRFCookie } from "../api/csrf";
import { getCurrentUser, login as loginApi, logout as logoutApi, refresh as refreshApi, signUp as signUpApi } from "../api/user";
import type { AuthUser, LoginInput, SignUpInput } from "../types/user";

interface AuthContextValue { user: AuthUser | null; accessToken: string | null; isAuthenticated: boolean; isLoading: boolean; authExpired: boolean; signUp: (input: SignUpInput) => Promise<void>; login: (input: LoginInput) => Promise<void>; logout: () => Promise<void>; clearAuth: () => void; }
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
        if (active) setUser(currentUser);
      } catch {
        if (active) clearAuth();
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
      setUser(response.user);
    } catch (error: unknown) {
      clearAuth();
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  const login = async (input: LoginInput): Promise<void> => {
    setIsLoading(true);
    try {
      const response = await loginApi(input);
      setAuthExpired(false);
      setAccessToken(response.access_token);
      setClientAccessToken(response.access_token);
      setUser(await getCurrentUser());
    } catch (error: unknown) {
      clearAuth();
      throw error;
    } finally { setIsLoading(false); }
  };

  const logout = async (): Promise<void> => {
    try {
      await logoutApi();
    } finally {
      clearAuth();
    }
  };

  const value = useMemo(() => ({ user, accessToken, isAuthenticated: user !== null && accessToken !== null, isLoading, authExpired, signUp, login, logout, clearAuth }), [user, accessToken, isLoading, authExpired]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within an AuthProvider");
  return context;
}
