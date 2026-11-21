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
import { useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { User, LoginRequest, RegisterRequest, InstanceInfo } from "@/api";
import {
  advanceAuthenticationAttemptRevision,
  advanceAuthenticationSubjectRevision,
  captureAuthenticationSubjectRevision,
  isAuthenticationAttemptRevisionCurrent,
  isAuthenticationSubjectRevisionCurrent,
  subscribeAuthenticationSubjectTermination,
  subscribeAuthenticationTokenReplacement,
  terminateCurrentAuthenticationSubject,
} from "@/utils/authenticationSubjectRevision";

export type AuthenticationCompletion =
  | { readonly status: "authenticated" }
  | { readonly status: "two_factor_required" }
  | { readonly status: "stale" };

export type WebAuthnAuthentication = Readonly<{
  token: string;
  user: User;
}>;

const AUTHENTICATED_COMPLETION = { status: "authenticated" } as const;
const TWO_FACTOR_REQUIRED_COMPLETION = {
  status: "two_factor_required",
} as const;
const STALE_COMPLETION = { status: "stale" } as const;

function requireAuthenticationData<Data>(data: Data | undefined): Data {
  if (data === undefined) {
    throw new Error("Authentication response did not include data");
  }
  return data;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  twoFactorPending: boolean;
  tempToken: string | null;
  login: (data: LoginRequest) => Promise<AuthenticationCompletion>;
  loginWithWebAuthn: (
    authenticate: () => Promise<WebAuthnAuthentication>,
  ) => Promise<AuthenticationCompletion>;
  register: (data: RegisterRequest) => Promise<AuthenticationCompletion>;
  logout: () => void;
  setExternalToken: (jwt: string) => void;
  beginExternalTwoFactor: (pendingTwoFactorToken: string) => void;
  verifyTwoFactor: (code: string) => Promise<AuthenticationCompletion>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(
    () => localStorage.getItem("token"),
  );
  const [isLoading, setIsLoading] = useState(
    () => !!localStorage.getItem("token"),
  );
  const [twoFactorPending, setTwoFactorPending] = useState(false);
  const [tempToken, setTempToken] = useState<string | null>(null);

  const retireAuthenticationSubject = useCallback(() => {
    // Subject transitions must remove account-scoped cache data and any pending authentication challenge.
    queryClient.clear();
    setUser(null);
    setTwoFactorPending(false);
    setTempToken(null);
  }, [queryClient]);

  const commitAuthenticationSubject = useCallback(() => {
    // A successful commit starts a new subject lifetime, invalidating work begun while the request was in flight.
    advanceAuthenticationSubjectRevision();
    retireAuthenticationSubject();
  }, [retireAuthenticationSubject]);

  const handleAuthenticationSubjectTermination = useCallback(() => {
    retireAuthenticationSubject();
    setIsLoading(false);
    setToken(null);
  }, [retireAuthenticationSubject]);

  useEffect(
    () =>
      subscribeAuthenticationSubjectTermination(
        handleAuthenticationSubjectTermination,
      ),
    [handleAuthenticationSubjectTermination],
  );

  useEffect(() => subscribeAuthenticationTokenReplacement(setToken), []);

  useEffect(() => {
    if (!token) {
      return;
    }
    let cancelled = false;
    const validationRevision = captureAuthenticationSubjectRevision();
    const validationIsCurrent = () =>
      !cancelled && isAuthenticationSubjectRevisionCurrent(validationRevision);
    api.auth
      .getAuth()
      .then((res) => {
        if (validationIsCurrent() && res.data) {
          setUser(res.data);
        }
      })
      .catch(() => {
        if (validationIsCurrent()) {
          terminateCurrentAuthenticationSubject();
        }
      })
      .finally(() => {
        if (validationIsCurrent()) {
          setIsLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [token]);

  const login = useCallback(
    async (data: LoginRequest): Promise<AuthenticationCompletion> => {
      const revision = advanceAuthenticationAttemptRevision();
      try {
        const res = await api.auth.loginCreate(data);
        if (!isAuthenticationAttemptRevisionCurrent(revision)) {
          return STALE_COMPLETION;
        }
        const auth = requireAuthenticationData(res.data);
        commitAuthenticationSubject();
        setIsLoading(false);
        if (auth.requires_two_factor) {
          setTwoFactorPending(true);
          setTempToken(auth.temp_token ?? null);
          setUser(auth.user ?? null);
          localStorage.removeItem("token");
          setToken(null);
          return TWO_FACTOR_REQUIRED_COMPLETION;
        }
        localStorage.setItem("token", auth.token);
        setToken(auth.token);
        setUser(auth.user);
        return AUTHENTICATED_COMPLETION;
      } catch (error) {
        if (!isAuthenticationAttemptRevisionCurrent(revision)) {
          return STALE_COMPLETION;
        }
        setIsLoading(false);
        throw error;
      }
    },
    [commitAuthenticationSubject],
  );

  const loginWithWebAuthn = useCallback(
    async (
      authenticate: () => Promise<WebAuthnAuthentication>,
    ): Promise<AuthenticationCompletion> => {
      const revision = advanceAuthenticationAttemptRevision();
      try {
        const auth = await authenticate();
        if (!isAuthenticationAttemptRevisionCurrent(revision)) {
          return STALE_COMPLETION;
        }
        commitAuthenticationSubject();
        localStorage.setItem("token", auth.token);
        setIsLoading(false);
        setToken(auth.token);
        setUser(auth.user);
        return AUTHENTICATED_COMPLETION;
      } catch (error) {
        if (!isAuthenticationAttemptRevisionCurrent(revision)) {
          return STALE_COMPLETION;
        }
        setIsLoading(false);
        throw error;
      }
    },
    [commitAuthenticationSubject],
  );

  const register = useCallback(
    async (data: RegisterRequest): Promise<AuthenticationCompletion> => {
      const revision = advanceAuthenticationAttemptRevision();
      try {
        const res = await api.auth.registerCreate(data);
        if (!isAuthenticationAttemptRevisionCurrent(revision)) {
          return STALE_COMPLETION;
        }
        const auth = requireAuthenticationData(res.data);
        commitAuthenticationSubject();
        localStorage.setItem("token", auth.token);
        setIsLoading(false);
        setToken(auth.token);
        setUser(auth.user);
        return AUTHENTICATED_COMPLETION;
      } catch (error) {
        if (!isAuthenticationAttemptRevisionCurrent(revision)) {
          return STALE_COMPLETION;
        }
        setIsLoading(false);
        throw error;
      }
    },
    [commitAuthenticationSubject],
  );

  const logout = useCallback(() => {
    terminateCurrentAuthenticationSubject();
  }, []);

  const verifyTwoFactor = useCallback(
    async (code: string): Promise<AuthenticationCompletion> => {
      if (!tempToken) {
        throw new Error("No temp token available");
      }
      const revision = advanceAuthenticationAttemptRevision();
      try {
        const res = await api.auth["2FaVerifyCreate"]({
          temp_token: tempToken,
          code,
        });
        if (!isAuthenticationAttemptRevisionCurrent(revision)) {
          return STALE_COMPLETION;
        }
        const auth = requireAuthenticationData(res.data);
        commitAuthenticationSubject();
        localStorage.setItem("token", auth.token);
        setIsLoading(false);
        setToken(auth.token);
        setUser(auth.user);
        return AUTHENTICATED_COMPLETION;
      } catch (error) {
        if (!isAuthenticationAttemptRevisionCurrent(revision)) {
          return STALE_COMPLETION;
        }
        setIsLoading(false);
        throw error;
      }
    },
    [tempToken, commitAuthenticationSubject],
  );

  const setExternalToken = useCallback(
    (jwt: string) => {
      if (jwt === token) {
        return;
      }
      advanceAuthenticationSubjectRevision();
      retireAuthenticationSubject();
      localStorage.setItem("token", jwt);
      setIsLoading(true);
      setToken(jwt);
    },
    [token, retireAuthenticationSubject],
  );

  // OAuth sign-in for an account with 2FA comes back with a pending token
  // instead of a session. Enter the same state a password login would be in
  // while it waits for the code.
  const beginExternalTwoFactor = useCallback(
    (pendingTwoFactorToken: string) => {
      advanceAuthenticationAttemptRevision();
      commitAuthenticationSubject();
      localStorage.removeItem("token");
      setIsLoading(false);
      setToken(null);
      setTwoFactorPending(true);
      setTempToken(pendingTwoFactorToken);
    },
    [commitAuthenticationSubject],
  );

  const value = useMemo(
    () => ({
      user,
      token,
      isAuthenticated: !!user,
      isLoading,
      twoFactorPending,
      tempToken,
      login,
      loginWithWebAuthn,
      register,
      logout,
      setExternalToken,
      beginExternalTwoFactor,
      verifyTwoFactor,
    }),
    [
      user,
      token,
      isLoading,
      twoFactorPending,
      tempToken,
      login,
      loginWithWebAuthn,
      register,
      logout,
      setExternalToken,
      beginExternalTwoFactor,
      verifyTwoFactor,
    ],
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
