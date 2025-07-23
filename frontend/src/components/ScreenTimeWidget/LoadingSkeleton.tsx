export function LoadingSkeleton() {
  return (
    <>
      {Array.from({ length: 3 }).map((_, index) => (
        <div key={index} className="flex items-center justify-between">
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <div className="w-2 h-2 rounded-full bg-muted"></div>
            <div className="h-4 bg-muted rounded flex-1 animate-pulse"></div>
          </div>
          <div className="w-8 h-3 bg-muted rounded animate-pulse"></div>
        </div>
      ))}
    </>
  );
}
