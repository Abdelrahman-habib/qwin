import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { APP_CONFIG } from "@/constants/app";
import { tabs } from "@/constants/tabs";
import { ScrollArea } from "../ui/scroll-area";

export function AppTabs() {
  return (
    <div className="flex h-full">
      <Tabs
        className="flex w-full"
        orientation="vertical"
        defaultValue={APP_CONFIG.DEFAULT_TAB}
      >
        <TabsList className="flex flex-col justify-start h-full bg-transparent p-0">
          {tabs.map((tab) => (
            <TabsTrigger key={tab.id} value={tab.id}>
              <tab.icon className="size-5" />
              <span className="sr-only">{tab.name}</span>
            </TabsTrigger>
          ))}
        </TabsList>
        <div className="flex-1 border-s border">
          <ScrollArea
            className={
              "[&>[data-radix-scroll-area-viewport]]:max-h-[calc(100vh_-_3.8rem)]"
            }
          >
            <div className="h-full">
              {tabs.map((tab) => (
                <TabsContent key={tab.id} value={tab.id}>
                  <tab.content />
                </TabsContent>
              ))}
            </div>
          </ScrollArea>
        </div>
      </Tabs>
    </div>
  );
}
