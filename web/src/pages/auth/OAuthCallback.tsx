import { useEffect } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { Spin } from "antd";
import { useAuth } from "@/stores/auth";

export default function OAuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { setExternalToken } = useAuth();

  useEffect(() => {
    const linkToken = searchParams.get("link_token");
    if (linkToken) {
      navigate(`/auth/oauth-link?link_token=${encodeURIComponent(linkToken)}`, { replace: true });
      return;
    }

    const token = searchParams.get("token");
    if (token) {
      // 必须通过 setExternalToken 同步更新 AuthProvider 的 React 状态，
      // 否则 ProtectedRoute 在检查 isAuthenticated 时 user 仍为 null，会重定向回 /login。
      setExternalToken(token);
      navigate("/vaults", { replace: true });
    } else {
      navigate("/login", { replace: true });
    }
  }, [searchParams, navigate, setExternalToken]);

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <Spin size="large" />
    </div>
  );
}
