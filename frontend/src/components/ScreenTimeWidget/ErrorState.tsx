import { AlertCircle } from "lucide-react";

interface ErrorStateProps {
  error: string;
}

export function ErrorState({ error }: ErrorStateProps) {
  return (
    <div className="w-full h-full bg-background backdrop-blur-lg rounded-lg border border-destructive text-foreground overflow-hidden flex items-center justify-center">
      <div className="text-center p-4">
        <AlertCircle className="w-6 h-6 text-destructive mx-auto mb-2" />
        <p className="text-xs text-destructive">{error}</p>
      </div>
    </div>
  );
}
