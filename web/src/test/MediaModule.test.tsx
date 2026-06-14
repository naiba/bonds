import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import PhotosModule from "@/pages/contact/modules/PhotosModule";
import { ConfigProvider, App } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { api } from "@/api";
import type { Photo } from "@/api";
import { MemoryRouter } from "react-router-dom";

Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: vi.fn().mockImplementation((query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

vi.mock("@/api", () => {
  return {
    api: {
      contactPhotos: {
        contactsPhotosList: vi.fn(),
        contactsPhotosDelete: vi.fn(),
      },
    },
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => {
    return {
      t: (key: string) => {
        const translations: Record<string, string> = {
          "modules.photos.title": "Media",
          "modules.photos.upload_text": "Click or drag to upload photos and videos",
          "modules.photos.no_photos": "No media uploaded",
        };
        return translations[key] || key;
      },
    };
  },
}));

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
});

const renderComponent = () => {
  return render(
    <ConfigProvider>
      <App>
        <MemoryRouter>
          <QueryClientProvider client={queryClient}>
            <PhotosModule vaultId="1" contactId="1" />
          </QueryClientProvider>
        </MemoryRouter>
      </App>
    </ConfigProvider>
  );
};

describe("PhotosModule (Media UI Requirements)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    queryClient.clear();
  });

  it("should render Media module title and updated empty state", async () => {
    vi.mocked(api.contactPhotos.contactsPhotosList).mockResolvedValue({
      success: true,
      data: [],
      meta: { total: 0, per_page: 30, page: 1, total_pages: 1 },
      error: undefined,
    });

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText("No media uploaded")).toBeInTheDocument();
    });

    expect(screen.getByText("Media")).toBeInTheDocument();
    expect(screen.getByText("Click or drag to upload photos and videos")).toBeInTheDocument();
  });

  it("should render both image and video items correctly based on mime type", async () => {
    const mockData: Photo[] = [
      {
        id: 1,
        name: "test-image.jpg",
        size: 1024,
        mime_type: "image/jpeg",
        created_at: "2024-01-01T00:00:00Z",
      },
      {
        id: 2,
        name: "test-video.mp4",
        size: 2048,
        mime_type: "video/mp4",
        created_at: "2024-01-02T00:00:00Z",
      },
    ];

    vi.mocked(api.contactPhotos.contactsPhotosList).mockResolvedValue({
      success: true,
      data: mockData,
      meta: { total: 2, per_page: 30, page: 1, total_pages: 1 },
      error: undefined,
    });

    renderComponent();

    await waitFor(() => {
      expect(document.querySelectorAll("video").length).toBeGreaterThan(0);
    });

    const imageElements = document.querySelectorAll("img");
    const hasImageSrc = Array.from(imageElements).some((img) => img.src.includes("files/1/download"));
    expect(hasImageSrc).toBe(true);

    const videoElements = document.querySelectorAll("video");
    expect(videoElements.length).toBe(1);
    expect(videoElements[0]).toHaveAttribute("controls");

    const sourceElements = videoElements[0].querySelectorAll("source");
    const hasVideoSrc =
      videoElements[0].src.includes("files/2/download") ||
      (sourceElements.length > 0 && sourceElements[0].src.includes("files/2/download"));
    expect(hasVideoSrc).toBe(true);
  });
});
