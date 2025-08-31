import { create } from "zustand";
import { persist } from "zustand/middleware";

interface SidebarState {
  isOpen: boolean;
}
interface SidebarActions {
  toggleSidebar: () => void;
}
interface SidebarStore extends SidebarState, SidebarActions {}

export const useSidebarStore = create<SidebarStore>()(
  persist(
    (set) => ({
      isOpen: false,
      toggleSidebar: () => set((state) => ({ isOpen: !state.isOpen })),
    }),
    {
      name: "sidebar-storage",
    }
  )
);
