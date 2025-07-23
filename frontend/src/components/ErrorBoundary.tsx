import { Component, ErrorInfo, ReactNode } from "react";
import { AlertTriangle } from "lucide-react";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
  };

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("Uncaught error:", error, errorInfo);
  }

  public render() {
    if (this.state.hasError) {
      return (
        <div className="w-full h-full bg-black/80 backdrop-blur-sm rounded-lg border border-red-700/50 text-white overflow-hidden flex items-center justify-center">
          <div className="text-center p-4">
            <AlertTriangle className="w-8 h-8 text-red-400 mx-auto mb-2" />
            <h2 className="text-sm font-medium text-red-400 mb-1">
              Something went wrong
            </h2>
            <p className="text-xs text-gray-400">
              {this.state.error?.message || "An unexpected error occurred"}
            </p>
            <button
              onClick={() =>
                this.setState({ hasError: false, error: undefined })
              }
              className="mt-3 px-3 py-1 text-xs bg-red-500/20 hover:bg-red-500/30 rounded transition-colors"
            >
              Try again
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
