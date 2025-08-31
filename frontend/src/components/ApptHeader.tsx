import React, { useEffect } from "react";
import {
  ArrowLeftIcon,
  ArrowRightIcon,
  Copy,
  MessageSquare,
  Calendar,
  SidebarCloseIcon,
  SidebarOpenIcon,
  Square,
  X,
} from "lucide-react";
import {
  WindowMinimise,
  Quit,
  WindowToggleMaximise,
  WindowIsMaximised,
} from "@wailsjs/runtime/runtime";
import { Button } from "@/components/ui/button";
import { useTitle } from "@/hooks/use-title";
import { useSidebar } from "@/hooks/use-sidebar";
import { cn } from "@/lib/utils";
import { APP_CONFIG } from "@/constants/app";
import { useCanGoBack, useRouter } from "@tanstack/react-router";

export function ApptHeader() {
  const router = useRouter();
  const canGoBack = useCanGoBack();
  const onBack = () => router.history.back();
  const onForward = () => router.history.forward();

  const { title } = useTitle();

  const {
    isOpen,
    toggleSidebar,
    isChatSidebarOpen,
    toggleChatSidebar,
    isCalendarSidebarOpen,
    toggleCalendarSidebar,
  } = useSidebar();

  const [isMaximised, setIsMaximised] = React.useState(false);

  const handleClose = () => {
    Quit();
  };

  const handleMinimize = () => {
    WindowMinimise();
  };

  const handleToggleMaximise = () => {
    setIsMaximised((prev) => !prev);
    WindowToggleMaximise();
  };

  useEffect(() => {
    try {
      WindowIsMaximised().then((isMaximised) => {
        setIsMaximised(isMaximised);
      });
    } catch (error) {
      console.error("Error checking if window is maximised:", error);
    }
  }, []);

  return (
    <div
      className="flex items-center justify-start border-b"
      style={{ "--wails-draggable": "drag" } as React.CSSProperties}
    >
      <div
        className={cn(
          "flex items-center gap-2 select-none px-3",
          APP_CONFIG.MAIN_SIDEBAR_MiN_WIDTH_CLASS
        )}
      >
        <img
          src="/assets/logo.svg"
          className="ms-1 size-8 invert contrast-0 pointer-events-none"
          alt="logo"
        />
        <span className="text-muted-foreground line-clamp-1 pointer-events-none">
          {APP_CONFIG.APP_NAME}
        </span>
      </div>
      <div className="flex items-center justify-between flex-1">
        <div
          className="flex items-center gap-1 border-l"
          style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
        >
          <Button
            onClick={toggleSidebar}
            variant="ghost"
            className="transition-colors"
          >
            {isOpen ? (
              <SidebarCloseIcon className="size-4" />
            ) : (
              <SidebarOpenIcon className="size-4" />
            )}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="transition-colors"
            onClick={onBack}
            disabled={!canGoBack}
          >
            <ArrowLeftIcon className="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="transition-colors"
            onClick={onForward}
          >
            <ArrowRightIcon className="size-4" />
          </Button>
        </div>
        {/* TODO: replace with search */}
        <div className="flex-1 text-center text-sm">{title}</div>
        <div
          className="flex items-center"
          style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
        >
          <Button
            onClick={toggleChatSidebar}
            variant="ghost"
            className={cn(
              "transition-colors hover:bg-primary/5 border-x border-x-transparent",
              isChatSidebarOpen && "bg-primary/5 border-x-primary/10"
            )}
          >
            <MessageSquare className="size-4" />
          </Button>
          <Button
            onClick={toggleCalendarSidebar}
            variant="ghost"
            className={cn(
              "transition-colors hover:bg-primary/5 border-x border-x-transparent",
              isCalendarSidebarOpen && "bg-primary/5 border-x-primary/10"
            )}
          >
            <Calendar className="size-4" />
          </Button>
          <Button
            variant="ghost"
            onClick={handleMinimize}
            className="transition-colors"
          >
            <div className="w-3 h-0.5 bg-muted-foreground" />
          </Button>
          <Button
            variant="ghost"
            onClick={handleToggleMaximise}
            className="transition-colors"
          >
            {isMaximised ? (
              <Copy className="size-3 rotate-90" />
            ) : (
              <Square className="size-3" />
            )}
          </Button>
          <Button
            variant="ghost"
            onClick={handleClose}
            className="hover:bg-destructive/50 transition-colors"
          >
            <X className="w-3 h-3 hover:text-destructive-foreground" />
          </Button>
        </div>
      </div>
    </div>
  );
}
