import { sidebarList } from "@/constants/sidebar";
import { Button } from "../ui/button";
import { Link, useLocation } from "@tanstack/react-router";
import { cn } from "@/lib/utils";

export function Sidebar() {
  const pathname = useLocation({
    select: (location) => location.pathname,
  });
  return (
    <div className="flex flex-col justify-start h-full bg-transparent p-0">
      {/* <label className="text-sm text-muted-foreground ms-1 mt-1">
        navigation
      </label> */}
      {sidebarList.map((tab) => (
        <Link to={tab.path} key={tab.id}>
          <Button
            variant="ghost"
            className={cn(
              "w-full flex gap-2 text-muted-foreground hover:bg-white/5 justify-start items-center border-s-4 border-transparent",
              pathname == tab.path && "bg-white/5 text-foreground  border-white"
            )}
          >
            <tab.icon className="size-5" />
            <span>{tab.name}</span>
          </Button>
        </Link>
      ))}
    </div>
  );
}
