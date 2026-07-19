import { createContext, useContext, useMemo, useState, type ReactNode } from "react";
import { setAccessToken as setClientAccessToken } from "../api/client";
import { login as loginApi } from "../api/user";
import type { AuthUser, LoginInput } from "../types/user";

interface AuthContextValue {
  user: AuthUser | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (input: LoginInput) => Promise<void>;
  clearAuth: () => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const login = async (input: LoginInput): Promise<void> => {
    setIsLoading(true);
    try {
      const response = await loginApi(input);
      setAccessToken(response.access_token);
      setClientAccessToken(response.access_token);
      setUser(response.user);
    } finally {
      setIsLoading(false);
    }
  };

  const clearAuth = (): void => {
    setAccessToken(null);
    setClientAccessToken(null);
    setUser(null);
  };

  const value = useMemo(() => ({ user, accessToken, isAuthenticated: user !== null && accessToken !== null, isLoading, login, clearAuth }), [user, accessToken, isLoading]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within an AuthProvider");
  return context;
}
