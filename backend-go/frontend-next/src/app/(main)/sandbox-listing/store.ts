'use client';

import { create } from 'zustand';

export type ListingStep = 1 | 2 | 3 | 4 | 5 | 6;

export interface SandboxListingState {
  // URL restore support
  candidateId: number | null;
  listingTaskId: number | null;
  approvalId: number | null;

  // Wizard state
  currentStep: ListingStep;

  // Navigation
  setStep: (step: ListingStep) => void;
  goNext: () => void;
  goBack: () => void;

  // Data
  setCandidateId: (id: number) => void;
  setListingTaskId: (id: number) => void;
  setApprovalId: (id: number) => void;

  // Reset
  reset: () => void;
}

export const useSandboxListingStore = create<SandboxListingState>((set) => ({
  candidateId: null,
  listingTaskId: null,
  approvalId: null,
  currentStep: 1,

  setStep: (step) => set({ currentStep: step }),
  goNext: () => set((s) => ({ currentStep: Math.min(6, s.currentStep + 1) as ListingStep })),
  goBack: () => set((s) => ({ currentStep: Math.max(1, s.currentStep - 1) as ListingStep })),

  setCandidateId: (id) => set({ candidateId: id }),
  setListingTaskId: (id) => set({ listingTaskId: id }),
  setApprovalId: (id) => set({ approvalId: id }),

  reset: () => set({
    candidateId: null,
    listingTaskId: null,
    approvalId: null,
    currentStep: 1,
  }),
}));
