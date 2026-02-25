import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { App as AntApp, ConfigProvider, theme as antTheme } from "antd";
import { useTheme } from "@/stores/theme";
import App from "./App.tsx";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

export default function ThemedApp() {
  const { resolvedTheme } = useTheme();

  return (
    <ConfigProvider
      theme={{
        algorithm:
          resolvedTheme === "dark"
            ? antTheme.darkAlgorithm
            : antTheme.defaultAlgorithm,
        token: {
          // Design system: Deep botanical & editorial palette
          colorPrimary: "#3b6347",
          colorPrimaryHover: "#4a7c59",
          colorSuccess: "#4b8f36",
          colorInfo: "#3b6347",
          colorWarning: "#d4a853",
          colorError: "#b84747",
          borderRadius: 12,
          borderRadiusLG: 16,
          fontFamily:
            "'DM Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif",
          fontFamilyCode:
            "'JetBrains Mono', 'SF Mono', SFMono-Regular, Menlo, Monaco, Consolas, monospace",
          fontSize: 14.5, // Slightly larger base for editorial feel
          fontSizeHeading1: 34,
          fontSizeHeading2: 28,
          fontSizeHeading3: 24,
          fontSizeHeading4: 20,
          fontSizeHeading5: 16,
          fontWeightStrong: 600,
          lineHeight: 1.65,
          motion: true,
          // Spacious padding
          paddingLG: 32,
          paddingMD: 24,
          paddingSM: 16,
          
          // Enhanced transparent layouts
          ...(resolvedTheme === "dark"
            ? {
                colorBgLayout: "transparent",
                colorBgContainer: "rgba(18, 22, 19, 0.75)",
                colorBgElevated: "rgba(24, 30, 26, 0.95)",
                colorTextBase: "rgba(255, 255, 255, 0.9)",
                colorBorder: "rgba(255, 255, 255, 0.08)",
              }
            : {
                colorBgLayout: "transparent",
                colorBgContainer: "rgba(255, 255, 255, 0.8)",
                colorBgElevated: "rgba(255, 255, 255, 0.98)",
                colorTextBase: "rgba(0, 0, 0, 0.85)",
                colorBorder: "rgba(0, 0, 0, 0.08)",
              }),
        },
        components: {
          Card: {
            borderRadiusLG: 16,
            paddingLG: 28,
          },
          Button: {
            borderRadius: 10,
            controlHeight: 40,
            fontWeight: 500,
          },
          Menu: {
            itemBorderRadius: 10,
            itemMarginInline: 6,
          },
          Table: {
            borderRadiusLG: 16,
            headerBg: resolvedTheme === "dark" ? "rgba(255,255,255,0.03)" : "rgba(0,0,0,0.02)",
            headerBorderRadius: 12,
          },
          Modal: {
            borderRadiusLG: 20,
          },
          Input: {
            borderRadius: 10,
            controlHeight: 42,
          },
          Select: {
            borderRadius: 10,
            controlHeight: 42,
          },
          Tabs: {
            itemSelectedColor: "#3b6347",
            inkBarColor: "#3b6347",
            titleFontSizeLG: 16,
          },
          Tag: {
            borderRadiusSM: 6,
          },
          Layout: {
            headerBg: resolvedTheme === "dark" ? "rgba(12, 14, 12, 0.85)" : "rgba(244, 245, 242, 0.85)",
          }
        },
      }}
    >
      <AntApp>
        <QueryClientProvider client={queryClient}>
          <App />
        </QueryClientProvider>
      </AntApp>
    </ConfigProvider>
  );
}
