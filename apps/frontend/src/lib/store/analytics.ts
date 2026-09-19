import { create } from "zustand";
import type { components } from "@/lib/api/gen";

type Window = components["schemas"]["Window"];

export interface AnalyticsState {
  window: Window;
  setWindow: (window: Window) => void;
}

export const useAnalyticsStore = create<AnalyticsState>()((set) => ({
  window: "week",
  setWindow: (window) => set({ window }),
}));
