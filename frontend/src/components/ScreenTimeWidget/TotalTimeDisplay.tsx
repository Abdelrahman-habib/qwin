import { ChevronRight, Clock } from "lucide-react";
import { formatTime } from "../../utils/timeFormatter";
import { Button } from "../ui/button";

interface TotalTimeDisplayProps {
  totalTime: number;
  isLoading: boolean;
}

export function TotalTimeDisplay({
  totalTime,
  isLoading,
}: TotalTimeDisplayProps) {
  return (
    <div className="px-3 py-1 border-b">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Clock className="w-4 h-4 text-primary" />
          {isLoading ? (
            <div className="w-16 h-6 bg-muted rounded animate-pulse"></div>
          ) : (
            <span className="text-lg font-bold text-primary">
              {formatTime(totalTime)}
            </span>
          )}
        </div>
        <div className="p-1">
          <Button
            variant="ghost"
            size="icon"
            className="text-muted-foreground rounded-full border border-white/5"
          >
            <ChevronRight className="ms-px" />
          </Button>
        </div>
      </div>
    </div>
  );
}
