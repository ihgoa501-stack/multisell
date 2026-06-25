import { create } from 'zustand';

interface AppState {
  sidebarCollapsed: boolean;
  copilotOpen: boolean;
  commandPaletteOpen: boolean;
  toolPanelOpen: boolean;
  activeTool: string | null;
  activityFeedOpen: boolean;
  unseenCount: number;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setCopilotOpen: (open: boolean) => void;
  setCommandPaletteOpen: (open: boolean) => void;
  toggleToolPanel: () => void;
  setActiveTool: (tool: string | null) => void;
  toggleActivityFeed: () => void;
  markActivitiesRead: () => void;
}

export const useAppStore = create<AppState>((set) => ({
  sidebarCollapsed: false,
  copilotOpen: false,
  commandPaletteOpen: false,
  toolPanelOpen: false,
  activeTool: null,
  activityFeedOpen: false,
  unseenCount: 3,
  toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
  setSidebarCollapsed: (collapsed: boolean) => set({ sidebarCollapsed: collapsed }),
  setCopilotOpen: (open: boolean) => set({ copilotOpen: open }),
  setCommandPaletteOpen: (open: boolean) => set({ commandPaletteOpen: open }),
  toggleToolPanel: () => set((state) => ({ toolPanelOpen: !state.toolPanelOpen })),
  setActiveTool: (tool: string | null) => set({ activeTool: tool }),
  toggleActivityFeed: () => set((state) => ({ activityFeedOpen: !state.activityFeedOpen })),
  markActivitiesRead: () => set({ unseenCount: 0 }),
}));
