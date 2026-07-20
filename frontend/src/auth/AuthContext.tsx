import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { setAccessToken as setClientAccessToken, setAuthStateListener } from "../api/client";
import { getCSRFCookie } from "../api/csrf";
import { getCurrentUser, login as loginApi, refresh as refreshApi } from "../api/user";
import type { AuthUser, LoginInput } from "../types/user";

interface AuthContextValue { user: AuthUser | null; accessToken: string | null; isAuthenticated: boolean; isLoading: boolean; login: (input: LoginInput) => Promise<void>; clearAuth: () => void; }
const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const clearAuth = (): void => { setAccessToken(null); setClientAccessToken(null); setUser(null); };

  useEffect(() => {
    setAuthStateListener((token) => { setAccessToken(token); if (token === null) setUser(null); });
    const restore = async (): Promise<void> => {
      try {
        if (!getCSRFCookie()) { clearAuth(); return; }
        const refreshed = await refreshApi();
        setAccessToken(refreshed.access_token);
        setClientAccessToken(refreshed.access_token);
        setUser(await getCurrentUser());
      } catch { clearAuth(); }
      finally { setIsLoading(false); }
    };
    void restore();
    return () => setAuthStateListener(null);
  }, []);

  const login = async (input: LoginInput): Promise<void> => {
    setIsLoading(true);
    try {
      const response = await loginApi(input);
      setAccessToken(response.access_token);
      setClientAccessToken(response.access_token);
      setUser(response.user);
    } finally { setIsLoading(false); }
  };

  const value = useMemo(() => ({ user, accessToken, isAuthenticated: user !== null && accessToken !== null, isLoading, login, clearAuth }), [user, accessToken, isLoading]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within an AuthProvider");
  return context;
}
