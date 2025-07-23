import { WidgetHeader } from "./WidgetHeader";
import { TitleProvider } from "@/contexts/TitleContext";

export function RootLayout({ children }: { children?: React.ReactNode }) {
  return (
    <TitleProvider>
      <div className="w-full h-full bg-transparent rounded-lg border text-foreground overflow-hidden">
        <WidgetHeader />
        {children}
      </div>
    </TitleProvider>
  );
}
