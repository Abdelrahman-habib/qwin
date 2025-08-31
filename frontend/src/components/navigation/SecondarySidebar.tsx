import { cn } from "@/lib/utils";
import { useSidebar } from "@/hooks/use-sidebar";
import { APP_CONFIG } from "@/constants/app";

import { PlusIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { useState } from "react";
import { formatDateRange } from "@/utils/dateTimeFormatter";
import { ScrollArea } from "../ui/scroll-area";

const events = [
  {
    title: "Team Sync Meeting",
    from: "2025-06-12T09:00:00",
    to: "2025-06-12T10:00:00",
  },
  {
    title: "Design Review",
    from: "2025-06-12T11:30:00",
    to: "2025-06-12T12:30:00",
  },
  {
    title: "Client Presentation",
    from: "2025-06-12T14:00:00",
    to: "2025-06-12T15:00:00",
  },
];

export function SecondarySidebar() {
  const { isCalendarSidebarOpen, isChatSidebarOpen } = useSidebar();

  if (!isCalendarSidebarOpen && !isChatSidebarOpen) return null;

  return (
    <div
      className={cn(
        "flex flex-col justify-start h-full bg-transparent p-0",
        APP_CONFIG.SECONDARY_SIDEBAR_MiN_WIDTH_CLASS
      )}
    >
      <ScrollArea className="h-auto max-h-[calc(100dvh_-_3rem)]">
        {isCalendarSidebarOpen && <CalanderSidebar />}
        {!isCalendarSidebarOpen && isChatSidebarOpen && <ChatSidebar />}
      </ScrollArea>
    </div>
  );
}

const CalanderSidebar = () => {
  const [date, setDate] = useState<Date | undefined>(new Date(2025, 5, 12));

  return (
    <>
      <div className="p-4">
        <Calendar
          mode="single"
          selected={date}
          onSelect={setDate}
          className="bg-transparent p-0 w-full"
          classNames={{
            day_button: "size-12 bg-primary/5 mx-px",
          }}
          required
        />
      </div>
      <div className="flex flex-col items-start gap-3 border-t px-4 !pt-4">
        <div className="flex w-full items-center justify-between px-1">
          <div className="text-sm font-medium">
            {date?.toLocaleDateString("en-US", {
              day: "numeric",
              month: "long",
              year: "numeric",
            })}
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="size-6"
            title="Add Event"
          >
            <PlusIcon />
            <span className="sr-only">Add Event</span>
          </Button>
        </div>
        <div className="flex w-full flex-col gap-2">
          {events.map((event) => (
            <div
              key={event.title}
              className="bg-white/5 after:bg-primary/70 relative rounded-md p-2 pl-6 text-sm after:absolute after:inset-y-2 after:left-2 after:w-1 after:rounded-full"
            >
              <div className="font-medium">{event.title}</div>
              <div className="text-muted-foreground text-xs">
                {formatDateRange(new Date(event.from), new Date(event.to))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </>
  );
};

const ChatSidebar = () => {
  return (
    <div className="flex flex-col items-start p-4 w-full h-full ">
      <div className="flex w-full items-center justify-between p-1">
        <div className="text-sm font-medium">Chat</div>
        <Button
          variant="ghost"
          size="icon"
          className="size-4"
          title="Add Event"
        >
          <PlusIcon />
          <span className="sr-only">Add Event</span>
        </Button>
      </div>
      <div className="flex w-full flex-col gap-2">
        {events.map((event) => (
          <div
            key={event.title}
            className="bg-white/5 after:bg-primary/70 relative rounded-md p-2 pl-6 text-sm after:absolute after:inset-y-2 after:left-2 after:w-1 after:rounded-full"
          >
            <div className="font-medium">{event.title}</div>
            <div className="text-muted-foreground text-xs">
              {formatDateRange(new Date(event.from), new Date(event.to))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
