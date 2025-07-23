import { ThemeProvider } from "@/components/theme-provider";
import SpotlightCard from "./components/ui/spotlight-card";
import { AppTabs } from "./components/navigation/app-tabs";
import { RootLayout } from "./components/layout";

function App() {
  return (
    <ThemeProvider defaultTheme="system" storageKey="vite-ui-theme">
      <SpotlightCard
        className="p-0 bg-transparent"
        spotlightColor="rgba(255, 255, 255, 0.07)"
      >
        <RootLayout>
          <AppTabs />
        </RootLayout>
      </SpotlightCard>
    </ThemeProvider>
  );
}

export default App;
