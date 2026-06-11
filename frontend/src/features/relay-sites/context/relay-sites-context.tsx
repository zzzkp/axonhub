'use client';

import { createContext, useContext, useState, type ReactNode } from 'react';
import type { RelaySite, RelaySiteAPIKey } from '../data/relay-sites';

interface RelaySitesContextType {
  isCreateDialogOpen: boolean;
  setIsCreateDialogOpen: (open: boolean) => void;
  isEditDialogOpen: boolean;
  setIsEditDialogOpen: (open: boolean) => void;
  isDeleteDialogOpen: boolean;
  setIsDeleteDialogOpen: (open: boolean) => void;
  isImportChannelDialogOpen: boolean;
  setIsImportChannelDialogOpen: (open: boolean) => void;
  isAPIKeysDialogOpen: boolean;
  setIsAPIKeysDialogOpen: (open: boolean) => void;
  isModelsDialogOpen: boolean;
  setIsModelsDialogOpen: (open: boolean) => void;
  isModelsAndTokensDialogOpen: boolean;
  setIsModelsAndTokensDialogOpen: (open: boolean) => void;
  isCheckinLogsDialogOpen: boolean;
  setIsCheckinLogsDialogOpen: (open: boolean) => void;
  isAnnouncementsDialogOpen: boolean;
  setIsAnnouncementsDialogOpen: (open: boolean) => void;
  editingRelaySite: RelaySite | null;
  setEditingRelaySite: (relaySite: RelaySite | null) => void;
  deletingRelaySite: RelaySite | null;
  setDeletingRelaySite: (relaySite: RelaySite | null) => void;
  importingRelaySite: RelaySite | null;
  setImportingRelaySite: (relaySite: RelaySite | null) => void;
  importingAPIKey: RelaySiteAPIKey | null;
  setImportingAPIKey: (apiKey: RelaySiteAPIKey | null) => void;
  managingRelaySite: RelaySite | null;
  setManagingRelaySite: (relaySite: RelaySite | null) => void;
  viewingModelsRelaySite: RelaySite | null;
  setViewingModelsRelaySite: (relaySite: RelaySite | null) => void;
  viewingCheckinLogsRelaySite: RelaySite | null;
  setViewingCheckinLogsRelaySite: (relaySite: RelaySite | null) => void;
  viewingAnnouncementsRelaySite: RelaySite | null;
  setViewingAnnouncementsRelaySite: (relaySite: RelaySite | null) => void;
}

const RelaySitesContext = createContext<RelaySitesContextType | undefined>(undefined);

export function useRelaySitesContext() {
  const context = useContext(RelaySitesContext);
  if (!context) {
    throw new Error('useRelaySitesContext must be used within RelaySitesProvider');
  }
  return context;
}

export default function RelaySitesProvider({ children }: { children: ReactNode }) {
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [isImportChannelDialogOpen, setIsImportChannelDialogOpen] = useState(false);
  const [isAPIKeysDialogOpen, setIsAPIKeysDialogOpen] = useState(false);
  const [isModelsDialogOpen, setIsModelsDialogOpen] = useState(false);
  const [isModelsAndTokensDialogOpen, setIsModelsAndTokensDialogOpen] = useState(false);
  const [isCheckinLogsDialogOpen, setIsCheckinLogsDialogOpen] = useState(false);
  const [isAnnouncementsDialogOpen, setIsAnnouncementsDialogOpen] = useState(false);
  const [editingRelaySite, setEditingRelaySite] = useState<RelaySite | null>(null);
  const [deletingRelaySite, setDeletingRelaySite] = useState<RelaySite | null>(null);
  const [importingRelaySite, setImportingRelaySite] = useState<RelaySite | null>(null);
  const [importingAPIKey, setImportingAPIKey] = useState<RelaySiteAPIKey | null>(null);
  const [managingRelaySite, setManagingRelaySite] = useState<RelaySite | null>(null);
  const [viewingModelsRelaySite, setViewingModelsRelaySite] = useState<RelaySite | null>(null);
  const [viewingCheckinLogsRelaySite, setViewingCheckinLogsRelaySite] = useState<RelaySite | null>(null);
  const [viewingAnnouncementsRelaySite, setViewingAnnouncementsRelaySite] = useState<RelaySite | null>(null);

  return (
    <RelaySitesContext.Provider
      value={{
        isCreateDialogOpen,
        setIsCreateDialogOpen,
        isEditDialogOpen,
        setIsEditDialogOpen,
        isDeleteDialogOpen,
        setIsDeleteDialogOpen,
        isImportChannelDialogOpen,
        setIsImportChannelDialogOpen,
        isAPIKeysDialogOpen,
        setIsAPIKeysDialogOpen,
        isModelsDialogOpen,
        setIsModelsDialogOpen,
        isModelsAndTokensDialogOpen,
        setIsModelsAndTokensDialogOpen,
        isCheckinLogsDialogOpen,
        setIsCheckinLogsDialogOpen,
        isAnnouncementsDialogOpen,
        setIsAnnouncementsDialogOpen,
        editingRelaySite,
        setEditingRelaySite,
        deletingRelaySite,
        setDeletingRelaySite,
        importingRelaySite,
        setImportingRelaySite,
        importingAPIKey,
        setImportingAPIKey,
        managingRelaySite,
        setManagingRelaySite,
        viewingModelsRelaySite,
        setViewingModelsRelaySite,
        viewingCheckinLogsRelaySite,
        setViewingCheckinLogsRelaySite,
        viewingAnnouncementsRelaySite,
        setViewingAnnouncementsRelaySite,
      }}
    >
      {children}
    </RelaySitesContext.Provider>
  );
}
