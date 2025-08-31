import { create } from "zustand";
import { persist } from "zustand/middleware";

interface SidebarState {
  isOpen: boolean;
  isChatSidebarOpen: boolean;
  isCalendarSidebarOpen: boolean;
}
interface SidebarActions {
  toggleSidebar: () => void;
  toggleChatSidebar: () => void;
  toggleCalendarSidebar: () => void;
}
interface SidebarStore extends SidebarState, SidebarActions {}

export const useSidebarStore = create<SidebarStore>()(
  persist(
    (set) => ({
      isOpen: true,
      isChatSidebarOpen: false,
      isCalendarSidebarOpen: false,
      toggleSidebar: () => set((state) => ({ isOpen: !state.isOpen })),
      toggleChatSidebar: () =>
        set((state) => ({
          isChatSidebarOpen: !state.isChatSidebarOpen,
          isCalendarSidebarOpen: false,
        })),
      toggleCalendarSidebar: () =>
        set((state) => ({
          isCalendarSidebarOpen: !state.isCalendarSidebarOpen,
          isChatSidebarOpen: false,
        })),
    }),
    {
      name: "sidebar-storage",
    }
  )
);
