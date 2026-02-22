import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import type { ReactNode } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { api } from "@/api";
import type { User, LoginRequest, RegisterRequest, InstanceInfo } from "@/api";

interface AuthContextType {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (data: LoginRequest) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(
    () => localStorage.getItem("token"),
  );
  const [isLoading, setIsLoading] = useState(
    () => !!localStorage.getItem("token"),
  );

  useEffect(() => {
    if (!token) {
      return;
    }
    let cancelled = false;
    api.auth
      .getAuth()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      .then((res: any) => {
        if (!cancelled && res.data) {
          setUser(res.data);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setToken(null);
          setUser(null);
          localStorage.removeItem("token");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [token]);

  const login = useCallback(async (data: LoginRequest) => {
    const res = await api.auth.loginCreate(data);
    const auth = res.data!;
    localStorage.setItem("token", auth.token);
    setToken(auth.token);
    setUser(auth.user);
  }, []);

  const register = useCallback(async (data: RegisterRequest) => {
    const res = await api.auth.registerCreate(data);
    const auth = res.data!;
    localStorage.setItem("token", auth.token);
    setToken(auth.token);
    setUser(auth.user);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem("token");
    setToken(null);
    setUser(null);
  }, []);

  const value = useMemo(
    () => ({
      user,
      token,
      isAuthenticated: !!user,
      isLoading,
      login,
      register,
      logout,
    }),
    [user, token, isLoading, login, register, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading, user } = useAuth();
  const location = useLocation();
  const [instanceInfo, setInstanceInfo] = useState<InstanceInfo | null>(null);
  const [loadingInstanceInfo, setLoadingInstanceInfo] = useState(true);

  useEffect(() => {
    let cancelled = false;
    api.instance
      .infoList()
      .then((res) => {
        if (!cancelled) {
          setInstanceInfo(res.data || {});
        }
      })
      .catch(() => {
        if (!cancelled) {
          setInstanceInfo({});
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingInstanceInfo(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (isLoading || loadingInstanceInfo) return null;
  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  if (user && !user.email_verified_at && instanceInfo?.require_email_verification) {
    return <Navigate to="/verify-email" replace />;
  }
  return <>{children}</>;
}
