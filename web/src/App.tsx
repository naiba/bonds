import { lazy, Suspense, useEffect, useState } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { Alert, Spin } from "antd";
import { AuthProvider, ProtectedRoute } from "@/stores/auth";
import Layout from "@/components/Layout";
import client from "@/api/client";

// Auth pages
const Login = lazy(() => import("@/pages/auth/Login"));
const Register = lazy(() => import("@/pages/auth/Register"));

// Vault pages
const VaultList = lazy(() => import("@/pages/vault/VaultList"));
const VaultCreate = lazy(() => import("@/pages/vault/VaultCreate"));
const VaultDetail = lazy(() => import("@/pages/vault/VaultDetail"));
const JournalList = lazy(() => import("@/pages/vault/JournalList"));
const JournalDetail = lazy(() => import("@/pages/vault/JournalDetail"));
const PostDetail = lazy(() => import("@/pages/vault/PostDetail"));
const GroupList = lazy(() => import("@/pages/vault/GroupList"));
const GroupDetail = lazy(() => import("@/pages/vault/GroupDetail"));
const VaultTasks = lazy(() => import("@/pages/vault/VaultTasks"));
const VaultFiles = lazy(() => import("@/pages/vault/VaultFiles"));
const VaultCalendar = lazy(() => import("@/pages/vault/VaultCalendar"));
const VaultReports = lazy(() => import("@/pages/vault/VaultReports"));
const VaultFeed = lazy(() => import("@/pages/vault/VaultFeed"));

// Contact pages
const ContactList = lazy(() => import("@/pages/contact/ContactList"));
const ContactCreate = lazy(() => import("@/pages/contact/ContactCreate"));
const ContactDetail = lazy(() => import("@/pages/contact/ContactDetail"));

// Settings pages
const Settings = lazy(() => import("@/pages/settings/Settings"));
const Preferences = lazy(() => import("@/pages/settings/Preferences"));
const Notifications = lazy(() => import("@/pages/settings/Notifications"));
const Personalize = lazy(() => import("@/pages/settings/Personalize"));
const Users = lazy(() => import("@/pages/settings/Users"));
const TwoFactor = lazy(() => import("@/pages/settings/TwoFactor"));
const Invitations = lazy(() => import("@/pages/settings/Invitations"));

// Public auth pages
const AcceptInvite = lazy(() => import("@/pages/auth/AcceptInvite"));
const OAuthCallback = lazy(() => import("@/pages/auth/OAuthCallback"));

function PageLoader() {
  return (
    <div
      style={{
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        minHeight: 320,
      }}
    >
      <Spin size="large" />
    </div>
  );
}

export default function App() {
  const [announcement, setAnnouncement] = useState("");

  useEffect(() => {
    client
      .get<{ data: { content: string } }>("/announcement")
      .then((res) => setAnnouncement(res.data.data?.content ?? ""))
      .catch(() => {});
  }, []);

  return (
    <BrowserRouter>
      <AuthProvider>
        {announcement && (
          <Alert
            message={announcement}
            type="warning"
            banner
            closable
            style={{ borderRadius: 0 }}
          />
        )}
        <Suspense fallback={<PageLoader />}>
          <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/accept-invite" element={<AcceptInvite />} />
          <Route path="/auth/callback" element={<OAuthCallback />} />

          <Route
            element={
              <ProtectedRoute>
                <Layout />
              </ProtectedRoute>
            }
          >
            <Route path="/vaults" element={<VaultList />} />
            <Route path="/vaults/create" element={<VaultCreate />} />
            <Route path="/vaults/:id" element={<VaultDetail />} />
            <Route path="/vaults/:id/contacts" element={<ContactList />} />
            <Route
              path="/vaults/:id/contacts/create"
              element={<ContactCreate />}
            />
            <Route
              path="/vaults/:id/contacts/:contactId"
              element={<ContactDetail />}
            />
            <Route path="/vaults/:id/journals" element={<JournalList />} />
            <Route
              path="/vaults/:id/journals/:journalId"
              element={<JournalDetail />}
            />
            <Route
              path="/vaults/:id/journals/:journalId/posts/:postId"
              element={<PostDetail />}
            />
            <Route path="/vaults/:id/groups" element={<GroupList />} />
            <Route
              path="/vaults/:id/groups/:groupId"
              element={<GroupDetail />}
            />
            <Route path="/vaults/:id/tasks" element={<VaultTasks />} />
            <Route path="/vaults/:id/files" element={<VaultFiles />} />
            <Route path="/vaults/:id/calendar" element={<VaultCalendar />} />
            <Route path="/vaults/:id/reports" element={<VaultReports />} />
            <Route path="/vaults/:id/feed" element={<VaultFeed />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/settings/preferences" element={<Preferences />} />
            <Route
              path="/settings/notifications"
              element={<Notifications />}
            />
            <Route path="/settings/personalize" element={<Personalize />} />
            <Route path="/settings/users" element={<Users />} />
            <Route path="/settings/2fa" element={<TwoFactor />} />
            <Route path="/settings/invitations" element={<Invitations />} />
          </Route>

          <Route path="/" element={<Navigate to="/vaults" replace />} />
          <Route path="*" element={<Navigate to="/vaults" replace />} />
        </Routes>
        </Suspense>
      </AuthProvider>
    </BrowserRouter>
  );
}
