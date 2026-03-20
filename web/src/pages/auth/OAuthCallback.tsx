import { useEffect } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { Spin } from "antd";

export default function OAuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  useEffect(() => {
    // OAuth account-binding flow: link_token means the provider is not yet
    // associated with any account — redirect to the dedicated linking page.
    const linkToken = searchParams.get("link_token");
    if (linkToken) {
      navigate(`/auth/oauth-link?link_token=${encodeURIComponent(linkToken)}`, { replace: true });
      return;
    }

    const token = searchParams.get("token");
    if (token) {
      localStorage.setItem("token", token);
      navigate("/vaults", { replace: true });
    } else {
      navigate("/login", { replace: true });
    }
  }, [searchParams, navigate]);

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
