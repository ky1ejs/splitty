import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { useMutation, useQuery } from "urql";
import { MeQuery, VerifyPasscodeMutation } from "../graphql/operations";
import { clearTokens, getAccessToken, setTokens } from "./token-store";

interface User {
  id: string;
  email: string;
  displayName: string;
}

interface AuthContextValue {
  isAuthenticated: boolean;
  user: User | null;
  loading: boolean;
  login: (email: string, code: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const [meResult] = useQuery({ query: MeQuery, pause: !getAccessToken() });
  const [, verifyPasscode] = useMutation(VerifyPasscodeMutation);

  useEffect(() => {
    if (!getAccessToken()) {
      setLoading(false);
      return;
    }
    if (meResult.fetching) return;
    if (meResult.data?.me) {
      setUser(meResult.data.me);
    } else {
      // Tokens exist but Me returned no user (expired/invalid tokens
      // that couldn't be refreshed). Clear stale tokens and reset user.
      clearTokens();
      setUser(null);
    }
    setLoading(false);
  }, [meResult.fetching, meResult.data]);

  const login = useCallback(
    async (email: string, code: string) => {
      const result = await verifyPasscode({ email, code });
      if (result.error) {
        throw new Error(result.error.message);
      }
      const data = result.data?.verifyPasscode;
      if (!data) {
        throw new Error("No response from server");
      }
      setTokens(data.accessToken, data.refreshToken);
      setUser(data.user);
    },
    [verifyPasscode],
  );

  const logout = useCallback(() => {
    clearTokens();
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated: user !== null,
        user,
        loading,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
