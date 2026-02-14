import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import AcceptInvite from "@/pages/auth/AcceptInvite";

vi.mock("@/api/invitations", () => ({
  invitationsApi: {
    accept: vi.fn(),
  },
}));

function renderAcceptInvite(token = "test123") {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={[`/accept-invite?token=${token}`]}>
          <AcceptInvite />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("AcceptInvite", () => {
  it("renders accept invitation title", () => {
    renderAcceptInvite();
    expect(screen.getByText("Accept Invitation")).toBeInTheDocument();
  });

  it("renders first name input", () => {
    renderAcceptInvite();
    expect(screen.getByPlaceholderText("First Name")).toBeInTheDocument();
  });

  it("renders password input", () => {
    renderAcceptInvite();
    expect(screen.getByPlaceholderText("Password")).toBeInTheDocument();
  });

  it("renders submit button", () => {
    renderAcceptInvite();
    expect(
      screen.getByRole("button", { name: /create account/i }),
    ).toBeInTheDocument();
  });

  it("renders last name input", () => {
    renderAcceptInvite();
    expect(screen.getByPlaceholderText("Last Name")).toBeInTheDocument();
  });
});
