import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App as AntApp, ConfigProvider, Form } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import GeocodingSettings from "@/pages/admin/GeocodingSettings";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  updateSettings: vi.fn(),
  updateProvider: vi.fn(),
  deleteProvider: vi.fn(),
}));

vi.mock("@/api", () => ({
  api: {
    admin: {
      geocodingList: mocks.list,
      geocodingUpdate: mocks.updateSettings,
      geocodingProvidersUpdate: mocks.updateProvider,
      geocodingProvidersDelete: mocks.deleteProvider,
    },
  },
}));

const catalog = {
  active_provider: "locationiq",
  precision: "exact",
  providers: [
    {
      id: "nominatim",
      name: "Nominatim",
      configured: true,
      has_stored_config: false,
      supports_autocomplete: false,
      notice: "public_nominatim",
      fields: [],
      config: {},
      attribution: [
        {
          label: "© OpenStreetMap contributors",
          url: "https://www.openstreetmap.org/copyright",
        },
      ],
    },
    {
      id: "locationiq",
      name: "LocationIQ",
      configured: true,
      has_stored_config: true,
      supports_autocomplete: true,
      fields: [
        { key: "api_key", type: "password", required: true, secret: true },
      ],
      config: { api_key: "***" },
      attribution: [
        {
          label: "Search by LocationIQ.com",
          url: "https://locationiq.com/attribution",
        },
      ],
    },
    {
      id: "geoapify",
      name: "Geoapify",
      configured: false,
      has_stored_config: false,
      supports_autocomplete: true,
      fields: [
        { key: "api_key", type: "password", required: true, secret: true },
      ],
      config: { api_key: "" },
      attribution: [
        { label: "Powered by Geoapify", url: "https://www.geoapify.com/" },
      ],
    },
    {
      id: "photon",
      name: "Photon",
      configured: true,
      has_stored_config: false,
      supports_autocomplete: true,
      notice: "public_demo",
      fields: [{ key: "base_url", type: "url", required: true, secret: false }],
      config: { base_url: "https://photon.komoot.io" },
      attribution: [
        { label: "Powered by Photon", url: "https://github.com/komoot/photon" },
      ],
    },
  ],
};

function renderGeocodingSettings() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <Form>
            <GeocodingSettings />
          </Form>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

describe("GeocodingSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.list.mockResolvedValue({ data: catalog });
    mocks.updateSettings.mockResolvedValue({ data: catalog });
    mocks.updateProvider.mockResolvedValue({ data: catalog });
    mocks.deleteProvider.mockResolvedValue({ data: catalog });
  });

  it("shows provider capabilities, public-service warnings, and the Photon default", async () => {
    renderGeocodingSettings();

    expect(
      await screen.findByText("Public Nominatim usage policy"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Public Photon is a fair-use demo"),
    ).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("https://photon.komoot.io"),
    ).toBeInTheDocument();
    expect(screen.getByText("Forward geocoding only")).toBeInTheDocument();
    expect(screen.getAllByText("Address autocomplete")).toHaveLength(3);
    expect(
      screen.getByText(
        "*** preserves the existing secret. Enter a new value to replace it.",
      ),
    ).toBeInTheDocument();
  });

  it("saves one structured configuration for a provider", async () => {
    const user = userEvent.setup();
    renderGeocodingSettings();

    const title = await screen.findByText("Geoapify", { selector: "strong" });
    const card = title.closest(".ant-card");
    expect(card).not.toBeNull();
    const apiKey = within(card as HTMLElement).getByLabelText("API key");
    await user.type(apiKey, "geoapify-key");
    await user.click(
      within(card as HTMLElement).getByRole("button", {
        name: /Save provider/,
      }),
    );

    await waitFor(() => {
      expect(mocks.updateProvider).toHaveBeenCalledWith("geoapify", {
        config: { api_key: "geoapify-key" },
      });
    });
  });

  it("can reset the provider's only stored configuration", async () => {
    const user = userEvent.setup();
    renderGeocodingSettings();

    const title = await screen.findByText("LocationIQ", { selector: "strong" });
    const card = title.closest(".ant-card");
    expect(card).not.toBeNull();
    await user.click(
      within(card as HTMLElement).getByRole("button", {
        name: /Reset configuration/,
      }),
    );
    await user.click(await screen.findByRole("button", { name: "OK" }));

    await waitFor(() =>
      expect(mocks.deleteProvider).toHaveBeenCalledWith("locationiq"),
    );
  });
});
