import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ConfigProvider, App as AntApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";

const mocks = vi.hoisted(() => ({
  overview: vi.fn(),
  addresses: vi.fn(),
  importantDates: vi.fn(),
  mood: vi.fn(),
  moodHistory: vi.fn(),
  map: vi.fn(),
  demographics: vi.fn(),
  interactions: vi.fn(),
  preferences: vi.fn(),
}));
const route = vi.hoisted(() => ({ vaultId: "vault-1" }));

vi.mock("@/api", () => ({
  api: {
    reports: {
      reportsOverviewList: mocks.overview,
      reportsAddressesList: mocks.addresses,
      reportsImportantDatesList: mocks.importantDates,
      reportsMoodTrackingEventsList: mocks.mood,
      reportsMapList: mocks.map,
      reportsDemographicsList: mocks.demographics,
      reportsInteractionsList: mocks.interactions,
      reportsAddressesCityDetail: vi.fn(),
      reportsAddressesCountryDetail: vi.fn(),
    },
    moodTracking: {
      moodTrackingEventsList: mocks.moodHistory,
    },
    preferences: { preferencesList: mocks.preferences },
  },
}));

vi.mock("react-router-dom", async () => ({
  ...(await vi.importActual<typeof import("react-router-dom")>(
    "react-router-dom",
  )),
  useParams: () => ({ id: route.vaultId }),
  useNavigate: () => vi.fn(),
}));

import VaultReports from "@/pages/vault/VaultReports";

function renderPage(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const wrap = (content: ReactNode) => (
    <ConfigProvider>
      <AntApp>
        <QueryClientProvider client={client}>
          <MemoryRouter>{content}</MemoryRouter>
        </QueryClientProvider>
      </AntApp>
    </ConfigProvider>
  );
  const result = render(wrap(ui));
  return {
    ...result,
    rerenderPage: (content: ReactNode) => result.rerender(wrap(content)),
  };
}

beforeEach(() => {
  route.vaultId = "vault-1";
  Object.values(mocks).forEach((mock) => mock.mockReset());
  mocks.preferences.mockResolvedValue({
    data: { name_order: "%first_name% %last_name%" },
  });
  mocks.overview.mockResolvedValue({
    data: {
      total_contacts: 2,
      total_addresses: 1,
      total_important_dates: 1,
      total_mood_entries: 0,
    },
  });
  mocks.addresses.mockResolvedValue({ data: [] });
  mocks.importantDates.mockResolvedValue({ data: [] });
  mocks.mood.mockResolvedValue({ data: [] });
  mocks.moodHistory.mockResolvedValue({ data: [] });
  mocks.map.mockResolvedValue({
    data: { total_addresses: 0, geocoded_count: 0, points: [], countries: [] },
  });
  mocks.demographics.mockResolvedValue({
    data: { total_contacts: 0, dimensions: [] },
  });
  mocks.interactions.mockResolvedValue({
    data: {
      total_activities: 0,
      total_interactions: 0,
      contact_count: 0,
      months: [],
      channels: [],
      most_frequent: [],
      gone_quiet: [],
    },
  });
});

describe("VaultReports", () => {
  it("keeps the sections the reports page has always had", async () => {
    renderPage(<VaultReports />);
    // These titles are also what the end-to-end test locates the page by.
    expect(await screen.findByText("Address Distribution")).toBeInTheDocument();
    expect(
      await screen.findByText("Important Dates Overview"),
    ).toBeInTheDocument();
    expect(await screen.findByText("Mood Trends")).toBeInTheDocument();
  });

  it("renders the map, cadence and demographics sections", async () => {
    renderPage(<VaultReports />);
    expect(await screen.findByText("Where they are")).toBeInTheDocument();
    expect(await screen.findByText("Staying in touch")).toBeInTheDocument();
    expect(await screen.findByText("Who they are")).toBeInTheDocument();
  });

  it("asks for two years of cadence by default", async () => {
    renderPage(<VaultReports />);
    await screen.findByText("Staying in touch");
    expect(mocks.interactions).toHaveBeenCalledWith("vault-1", { months: 24 });
  });

  it("does not show the previous vault's interactions while a new vault loads", async () => {
    mocks.interactions.mockImplementation((vaultId: string) => {
      if (vaultId === "vault-2") return new Promise(() => undefined);
      return Promise.resolve({
        data: {
          total_activities: 1,
          total_interactions: 1,
          interaction_types_configured: true,
          contact_count: 1,
          months: [],
          channels: [
            { activity_type_id: 1, label: "Call", count: 1, months: [] },
          ],
          most_frequent: [
            {
              contact_id: "contact-1",
              contact_name: "Vault One Contact",
              count: 1,
            },
          ],
          gone_quiet: [],
        },
      });
    });

    const view = renderPage(<VaultReports />);
    expect(await screen.findByText("Vault One Contact")).toBeInTheDocument();

    route.vaultId = "vault-2";
    view.rerenderPage(<VaultReports />);

    expect(screen.queryByText("Vault One Contact")).not.toBeInTheDocument();
    expect(mocks.interactions).toHaveBeenCalledWith("vault-2", { months: 24 });
  });

  it("explains an empty cadence chart instead of drawing nothing", async () => {
    mocks.interactions.mockResolvedValue({
      data: {
        total_activities: 7643,
        total_interactions: 0,
        contact_count: 0,
        months: [],
        channels: [],
        most_frequent: [],
        gone_quiet: [],
      },
    });
    renderPage(<VaultReports />);
    // A vault full of activities whose types are unflagged should be told why,
    // not shown a blank chart.
    expect(
      await screen.findByText("No activity type counts as an interaction"),
    ).toBeInTheDocument();
  });

  it("shows how many addresses are actually plotted", async () => {
    mocks.map.mockResolvedValue({
      data: {
        total_addresses: 379,
        geocoded_count: 12,
        points: [],
        countries: [
          {
            country: "United Kingdom",
            address_count: 227,
            contact_count: 180,
            geocoded: 12,
          },
        ],
      },
    });
    renderPage(<VaultReports />);
    expect(
      await screen.findByText(/12 of 379 addresses have coordinates/),
    ).toBeInTheDocument();
  });

  it("links the geocoding attribution beside plotted provider results", async () => {
    mocks.map.mockResolvedValue({
      data: {
        total_addresses: 1,
        geocoded_count: 1,
        points: [
          { address_id: 1, latitude: 51.5, longitude: -0.12, contacts: [] },
        ],
        countries: [
          {
            country: "United Kingdom",
            address_count: 1,
            contact_count: 1,
            geocoded: 1,
          },
        ],
        attribution: [
          {
            label: "© OpenStreetMap contributors",
            url: "https://www.openstreetmap.org/copyright",
          },
        ],
      },
    });

    renderPage(<VaultReports />);

    expect(
      await screen.findByRole("link", { name: "© OpenStreetMap contributors" }),
    ).toHaveAttribute("href", "https://www.openstreetmap.org/copyright");
  });

  it("shows mood notes and hours slept in the mood history", async () => {
    mocks.overview.mockResolvedValue({
      data: {
        total_contacts: 2,
        total_addresses: 1,
        total_important_dates: 1,
        total_mood_entries: 1,
      },
    });
    mocks.mood.mockResolvedValue({
      data: [{ parameter_label: "Good", hex_color: "#84cc16", count: 1 }],
    });
    mocks.moodHistory.mockResolvedValue({
      data: [
        {
          id: 9,
          rated_at: "2026-09-01T08:30:00Z",
          parameter_label: "Good",
          hex_color: "#84cc16",
          number_of_hours_slept: 7,
          note: "A calm morning",
        },
      ],
    });

    renderPage(<VaultReports />);

    expect(await screen.findByText("Mood History")).toBeInTheDocument();
    expect(await screen.findByText("A calm morning")).toBeInTheDocument();
    expect(screen.getByText("7 h")).toBeInTheDocument();
    expect(mocks.moodHistory).toHaveBeenCalledWith("vault-1");
  });
});
