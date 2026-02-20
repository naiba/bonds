import { useState, useEffect } from "react";
import { httpClient } from "@/api";

const AVATAR_COLORS = [
  "#f56a00", "#7265e6", "#ffbf00", "#00a2ae",
  "#87d068", "#1677ff", "#722ed1", "#eb2f96",
  "#fa8c16", "#13c2c2", "#2f54eb", "#52c41a",
];

function getAvatarColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

export default function ContactAvatar({
  vaultId,
  contactId,
  firstName,
  lastName,
  size = 34,
  updatedAt,
}: {
  vaultId: string;
  contactId: string;
  firstName?: string;
  lastName?: string;
  size?: number;
  updatedAt?: string;
}) {
  const [blobUrl, setBlobUrl] = useState<string | null>(null);

  const initials = `${(firstName ?? "").charAt(0)}${(lastName ?? "").charAt(0)}`.toUpperCase() || "?";
  const bgColor = getAvatarColor((firstName ?? "") + (lastName ?? ""));

  useEffect(() => {
    let revoke: string | null = null;
    let cancelled = false;

    httpClient.instance
      .get(`/vaults/${vaultId}/contacts/${contactId}/avatar`, {
        responseType: "blob",
        params: updatedAt ? { t: updatedAt } : undefined,
      })
      .then((response) => {
        if (cancelled) return;
        const blob = response.data as Blob;
        if (blob.size > 0) {
          const url = URL.createObjectURL(blob);
          revoke = url;
          setBlobUrl(url);
        }
      })
      .catch(() => {
        // Fall back to initials
      });

    return () => {
      cancelled = true;
      if (revoke) URL.revokeObjectURL(revoke);
    };
  }, [vaultId, contactId, updatedAt]);

  const containerStyle: React.CSSProperties = {
    width: size,
    height: size,
    borderRadius: "50%",
    backgroundColor: bgColor,
    color: "#fff",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontSize: Math.max(size * 0.38, 11),
    fontWeight: 600,
    flexShrink: 0,
    overflow: "hidden",
    letterSpacing: 0.5,
  };

  if (blobUrl) {
    return (
      <div style={containerStyle}>
        <img
          src={blobUrl}
          alt={initials}
          style={{ width: "100%", height: "100%", objectFit: "cover" }}
        />
      </div>
    );
  }

  return <div style={containerStyle}>{initials}</div>;
}
