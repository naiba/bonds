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
          colorPrimary: "#5b8c5a",
          colorSuccess: "#52c41a",
          colorInfo: "#5b8c5a",
          borderRadius: 8,
          fontFamily:
            "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
          motion: true,
        },
        components: {
          Card: {
            borderRadiusLG: 12,
          },
          Button: {
            borderRadius: 8,
            controlHeight: 36,
          },
          Menu: {
            itemBorderRadius: 8,
            itemMarginInline: 8,
          },
          Table: {
            borderRadiusLG: 12,
          },
          Modal: {
            borderRadiusLG: 12,
          },
          Input: {
            borderRadius: 8,
          },
          Select: {
            borderRadius: 8,
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
