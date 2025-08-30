import React, { useEffect } from "react";
import { Copy, Square, X } from "lucide-react";
import {
  WindowMinimise,
  Quit,
  WindowToggleMaximise,
  WindowIsMaximised,
} from "@wailsjs/runtime/runtime";
import { Button } from "@/components/ui/button";
import { useTitle } from "@/hooks/use-title";

export function ApptHeader() {
  const { title } = useTitle();
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
      className="flex items-center justify-between px-3 py-2 border-b"
      style={{ "--wails-draggable": "drag" } as React.CSSProperties}
    >
      <div className="flex-1 flex items-center gap-2 select-none cursor-move">
        <img
          src="/assets/logo.svg"
          className="ms-1 size-8 invert contrast-0 pointer-events-none"
          alt="logo"
        />
        <span className="text-muted-foreground line-clamp-1 pointer-events-none">
          {title}
        </span>
      </div>
      <div
        className="flex items-center gap-1"
        style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
      >
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
  );
}
