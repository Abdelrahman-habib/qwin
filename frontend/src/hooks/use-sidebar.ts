import { useSidebarStore } from "@/store/sidebar";
import { useShallow } from "zustand/react/shallow";

export const useSidebar = () =>
  useSidebarStore(
    useShallow((state) => {
      return {
        isOpen: state.isOpen,
        toggleSidebar: state.toggleSidebar,
      };
    })
  );
