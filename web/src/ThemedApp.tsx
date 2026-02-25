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
          // Design system: warm olive-sage palette with refined typographic hierarchy
          colorPrimary: "#4a7c59",
          colorSuccess: "#52c41a",
          colorInfo: "#4a7c59",
          colorWarning: "#d4a853",
          colorError: "#cf4f4f",
          borderRadius: 10,
          fontFamily:
            "'DM Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
          fontFamilyCode:
            "'JetBrains Mono', 'SF Mono', SFMono-Regular, Menlo, Monaco, Consolas, monospace",
          fontSize: 14,
          fontSizeHeading1: 32,
          fontSizeHeading2: 26,
          fontSizeHeading3: 22,
          fontSizeHeading4: 18,
          fontSizeHeading5: 16,
          fontWeightStrong: 600,
          lineHeight: 1.6,
          motion: true,
          // Refined spacing
          paddingLG: 28,
          paddingMD: 20,
          paddingSM: 14,
          // Warmer backgrounds depending on theme
          ...(resolvedTheme === "dark"
            ? {
                colorBgLayout: "#0f1210",
                colorBgContainer: "#171c18",
                colorBgElevated: "#1e241f",
              }
            : {
                colorBgLayout: "#f7f8f5",
                colorBgContainer: "#ffffff",
                colorBgElevated: "#ffffff",
              }),
        },
        components: {
          Card: {
            borderRadiusLG: 14,
            paddingLG: 24,
          },
          Button: {
            borderRadius: 10,
            controlHeight: 38,
            fontWeight: 500,
          },
          Menu: {
            itemBorderRadius: 10,
            itemMarginInline: 6,
          },
          Table: {
            borderRadiusLG: 14,
            headerBg: resolvedTheme === "dark" ? "#1e241f" : "#f0f2ed",
          },
          Modal: {
            borderRadiusLG: 16,
          },
          Input: {
            borderRadius: 10,
            controlHeight: 40,
          },
          Select: {
            borderRadius: 10,
            controlHeight: 40,
          },
          Tabs: {
            itemSelectedColor: "#4a7c59",
            inkBarColor: "#4a7c59",
          },
          Tag: {
            borderRadiusSM: 8,
          },
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
