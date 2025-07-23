import { useTitle } from "@/hooks/use-title";

export const LearningTracker = () => {
  const { setTitle } = useTitle();

  setTitle("Learning tracker");
  return (
    <div className="flex flex-col items-center justify-center">
      <div className="flex flex-col items-center justify-center">
        <h1 className="text-2xl font-bold">Learning Tracker</h1>
        <p className="text-gray-500">Coming soon...</p>
      </div>
    </div>
  );
};
