import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { z } from 'zod';
import { graphqlRequest } from '@/gql/graphql';
import { useErrorHandler } from '@/hooks/use-error-handler';
import {
  createRelaySiteInputSchema,
  importRelaySiteAPIKeyToChannelInputSchema,
  importRelaySitesBackupInputSchema,
  importedRelaySiteChannelSchema,
  relaySiteAnnouncementResultSchema,
  relaySiteAPIKeyConfigInputSchema,
  relaySiteBatchOperationResultSchema,
  relaySiteCheckinPageSchema,
  relaySiteCheckinLogSchema,
  relaySiteFormResultSchema,
  relaySiteSchema,
  relaySitesConnectionSchema,
  updateRelaySiteInputSchema,
  type CreateRelaySiteInput,
  type ImportedRelaySiteChannel,
  type ImportRelaySiteAPIKeyToChannelInput,
  type ImportRelaySitesBackupInput,
  type RelaySite,
  type RelaySiteAnnouncement,
  type RelaySiteAnnouncementResult,
  type RelaySiteAPIKey,
  type RelaySiteBatchOperationResult,
  type RelaySiteAPIKeyConfigInput,
  type RelaySiteCheckinPage,
  type RelaySiteCheckinLog,
  type RelaySiteFormResult,
  type RelaySiteModelPrice,
  type RelaySitesConnection,
  type UpdateRelaySiteInput,
} from './schema';

export { useUpdateChannelStatus } from '@/features/channels/data/channels';

export type { CreateRelaySiteInput, RelaySite, RelaySiteAnnouncement, RelaySiteAnnouncementResult, RelaySiteAPIKey, RelaySiteCheckinLog, RelaySiteModelPrice, RelaySitesConnection, UpdateRelaySiteInput };
export type { ImportedRelaySiteChannel, ImportRelaySiteAPIKeyToChannelInput, ImportRelaySitesBackupInput, RelaySiteAPIKeyConfigInput, RelaySiteBatchOperationResult, RelaySiteCheckinPage };

const RELAY_SITE_FIELDS = `
  id
  createdAt
  updatedAt
  name
  type
  baseURL
  status
  autoCheckinEnabled
  remark
  lastSyncedAt
  lastSyncError
  lastCheckinAt
  lastCheckinResult
  checkinPageURL
  externalCheckinPageURL
  hasUnreadAnnouncements
  displayCredential { authType token userId username password refreshToken tokenExpiresAt }
  apiKeys(first: 100, orderBy: { field: UPDATED_AT, direction: DESC }) {
    edges { node { id remoteID name status groupName quota usedQuota expiresAt syncedAt } }
    totalCount
  }
  groups(first: 100, orderBy: { field: UPDATED_AT, direction: DESC }) {
    edges { node { id name ratio syncedAt } }
    totalCount
  }
  balanceSnapshots(first: 3, orderBy: { field: UPDATED_AT, direction: DESC }) {
    edges { node { id balance unit pulledAt } }
    totalCount
  }
  modelPrices(first: 500, orderBy: { field: UPDATED_AT, direction: DESC }) {
    edges { node { id modelID promptPrice completionPrice billingUnit enableGroups quotaType modelPrice modelRatio completionRatio supportedEndpointTypes syncedAt } }
    totalCount
  }
  checkinLogs(first: 100, orderBy: { field: UPDATED_AT, direction: DESC }) {
    edges { node { id executedAt status message errorMessage } }
    totalCount
  }
  announcements(first: 100, orderBy: { field: UPDATED_AT, direction: DESC }) {
    edges { node { id remoteID type content extra publishedAt fetchedAt readAt } }
    totalCount
  }
`;

const RELAY_SITE_FORM_FIELDS = `
  id
  createdAt
  updatedAt
  name
  type
  baseURL
  status
  autoCheckinEnabled
  remark
  lastSyncedAt
  lastSyncError
  lastCheckinAt
  lastCheckinResult
  checkinPageURL
  externalCheckinPageURL
  hasUnreadAnnouncements
`;

const RELAY_SITE_ANNOUNCEMENT_FIELDS = `
  id
  name
  hasUnreadAnnouncements
  announcements(first: 100, orderBy: { field: UPDATED_AT, direction: DESC }) {
    edges { node { id remoteID type content extra publishedAt fetchedAt readAt } }
    totalCount
  }
`;

const RELAY_SITE_ANNOUNCEMENTS_QUERY = `
  query RelaySiteAnnouncements($id: ID!) {
    node(id: $id) {
      ... on RelaySite {
        ${RELAY_SITE_ANNOUNCEMENT_FIELDS}
      }
    }
  }
`;

const RELAY_SITES_QUERY = `
  query RelaySites($first: Int, $after: Cursor, $where: RelaySiteWhereInput, $orderBy: RelaySiteOrder) {
    relaySites(first: $first, after: $after, where: $where, orderBy: $orderBy) {
      edges { node { ${RELAY_SITE_FIELDS} } }
      pageInfo { hasNextPage hasPreviousPage startCursor endCursor }
      totalCount
    }
  }
`;

const CREATE_RELAY_SITE_MUTATION = `
  mutation CreateRelaySiteConfig($input: CreateRelaySiteConfigInput!) {
    createRelaySiteConfig(input: $input) { ${RELAY_SITE_FORM_FIELDS} }
  }
`;

const UPDATE_RELAY_SITE_MUTATION = `
  mutation UpdateRelaySiteConfig($id: ID!, $input: UpdateRelaySiteConfigInput!) {
    updateRelaySiteConfig(id: $id, input: $input) { ${RELAY_SITE_FORM_FIELDS} }
  }
`;

const DELETE_RELAY_SITE_MUTATION = `
  mutation DeleteRelaySiteConfig($id: ID!) {
    deleteRelaySiteConfig(id: $id) { id name status }
  }
`;

const EXPORT_RELAY_SITES_BACKUP_MUTATION = `
  mutation ExportRelaySitesBackup {
    exportRelaySitesBackup
  }
`;

const IMPORT_RELAY_SITES_BACKUP_MUTATION = `
  mutation ImportRelaySitesBackup($payload: String!) {
    importRelaySitesBackup(payload: $payload)
  }
`;

const SYNC_RELAY_SITE_MUTATION = `
  mutation SyncRelaySite($id: ID!) {
    syncRelaySite(id: $id) { ${RELAY_SITE_FIELDS} }
  }
`;

const SYNC_ALL_RELAY_SITES_MUTATION = `
  mutation SyncAllRelaySites {
    syncAllRelaySites { totalCount successCount failedCount failures { relaySiteID relaySiteName errorMessage } }
  }
`;

const CHECKIN_RELAY_SITE_MUTATION = `
  mutation CheckinRelaySite($id: ID!) {
    checkinRelaySite(id: $id) { id executedAt status message errorMessage }
  }
`;

const CHECKIN_ALL_RELAY_SITES_MUTATION = `
  mutation CheckinAllRelaySites {
    checkinAllRelaySites { totalCount successCount failedCount failures { relaySiteID relaySiteName errorMessage } }
  }
`;

const FAILED_RELAY_SITE_CHECKIN_PAGES_QUERY = `
  query FailedRelaySiteCheckinPages {
    failedRelaySiteCheckinPages { relaySiteID relaySiteName url }
  }
`;

const REFRESH_RELAY_SITE_ANNOUNCEMENTS_MUTATION = `
  mutation RefreshRelaySiteAnnouncements($id: ID!) {
    refreshRelaySiteAnnouncements(id: $id) { id }
  }
`;

const MARK_RELAY_SITE_ANNOUNCEMENTS_READ_MUTATION = `
  mutation MarkRelaySiteAnnouncementsRead($id: ID!) {
    markRelaySiteAnnouncementsRead(id: $id) { id }
  }
`;

const IMPORT_RELAY_SITE_API_KEY_TO_CHANNEL_MUTATION = `
  mutation ImportRelaySiteAPIKeyToChannel($relaySiteAPIKeyID: ID!, $input: ImportRelaySiteAPIKeyToChannelInput!) {
    importRelaySiteAPIKeyToChannel(relaySiteAPIKeyID: $relaySiteAPIKeyID, input: $input) {
      id
      name
      type
      baseURL
      status
      supportedModels
      defaultTestModel
    }
  }
`;

const RELAY_SITE_CHANNELS_QUERY = `
  query RelaySiteChannels($input: QueryChannelInput!) {
    queryChannels(input: $input) {
      edges {
        node {
          id
          status
          tags
          endpoints {
            apiFormat
            path
            baseURL
            transport
          }
        }
      }
    }
  }
`;

const CREATE_RELAY_SITE_API_KEY_MUTATION = `
  mutation CreateRelaySiteAPIKey($relaySiteID: ID!, $input: RelaySiteAPIKeyConfigInput!) {
    createRelaySiteAPIKey(relaySiteID: $relaySiteID, input: $input) { ${RELAY_SITE_FIELDS} }
  }
`;

const UPDATE_RELAY_SITE_API_KEY_MUTATION = `
  mutation UpdateRelaySiteAPIKey($relaySiteAPIKeyID: ID!, $input: RelaySiteAPIKeyConfigInput!) {
    updateRelaySiteAPIKey(relaySiteAPIKeyID: $relaySiteAPIKeyID, input: $input) { ${RELAY_SITE_FIELDS} }
  }
`;

const DELETE_RELAY_SITE_API_KEY_MUTATION = `
  mutation DeleteRelaySiteAPIKey($relaySiteAPIKeyID: ID!) {
    deleteRelaySiteAPIKey(relaySiteAPIKeyID: $relaySiteAPIKeyID) { ${RELAY_SITE_FIELDS} }
  }
`;

const CREATE_RELAY_SITE_API_KEYS_FOR_ALL_GROUPS_MUTATION = `
  mutation CreateRelaySiteAPIKeysForAllGroups($relaySiteID: ID!, $input: RelaySiteAPIKeyConfigInput!) {
    createRelaySiteAPIKeysForAllGroups(relaySiteID: $relaySiteID, input: $input) { ${RELAY_SITE_FIELDS} }
  }
`;

export function useRelaySites(variables?: Record<string, any>) {
  const { t } = useTranslation();
  const { handleError } = useErrorHandler();

  return useQuery({
    queryKey: ['relaySites', variables],
    queryFn: async () => {
      try {
        const data = await graphqlRequest<{ relaySites: RelaySitesConnection }>(RELAY_SITES_QUERY, variables);
        return relaySitesConnectionSchema.parse(data.relaySites);
      } catch (error) {
        handleError(error, t('common.errors.loadFailed'));
        throw error;
      }
    },
  });
}

export function useCreateRelaySite() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (input: CreateRelaySiteInput) => {
      try {
        const validatedInput = createRelaySiteInputSchema.parse(input);
        const data = await graphqlRequest<{ createRelaySiteConfig: RelaySiteFormResult }>(CREATE_RELAY_SITE_MUTATION, {
          input: validatedInput,
        });
        return relaySiteFormResultSchema.parse(data.createRelaySiteConfig);
      } catch (error) {
        handleError(error, { context: t('relaySites.dialogs.create.title') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      toast.success(t('common.messages.success'));
    },
  });
}

export function useUpdateRelaySite() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: UpdateRelaySiteInput }) => {
      try {
        const validatedInput = updateRelaySiteInputSchema.parse(input);
        const data = await graphqlRequest<{ updateRelaySiteConfig: RelaySiteFormResult }>(UPDATE_RELAY_SITE_MUTATION, {
          id,
          input: validatedInput,
        });
        return relaySiteFormResultSchema.parse(data.updateRelaySiteConfig);
      } catch (error) {
        handleError(error, { context: t('relaySites.dialogs.edit.title') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      toast.success(t('common.messages.success'));
    },
  });
}

export function useDeleteRelaySite() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (id: string) => {
      try {
        const data = await graphqlRequest<{ deleteRelaySiteConfig: Pick<RelaySite, 'id' | 'name' | 'status'> }>(
          DELETE_RELAY_SITE_MUTATION,
          { id }
        );
        return data.deleteRelaySiteConfig;
      } catch (error) {
        handleError(error, { context: t('relaySites.dialogs.delete.title') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      toast.success(t('relaySites.messages.deleteSuccess'));
    },
  });
}

export function useExportRelaySitesBackup() {
  const { t } = useTranslation();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async () => {
      try {
        const data = await graphqlRequest<{ exportRelaySitesBackup: string }>(EXPORT_RELAY_SITES_BACKUP_MUTATION);
        return data.exportRelaySitesBackup;
      } catch (error) {
        handleError(error, { context: t('relaySites.buttons.exportBackup') });
        throw error;
      }
    },
    onSuccess: () => {
      toast.success(t('relaySites.messages.exportBackupSuccess'));
    },
  });
}

export function useImportRelaySitesBackup() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (input: ImportRelaySitesBackupInput) => {
      try {
        const validatedInput = importRelaySitesBackupInputSchema.parse(input);
        const data = await graphqlRequest<{ importRelaySitesBackup: boolean }>(IMPORT_RELAY_SITES_BACKUP_MUTATION, validatedInput);
        return data.importRelaySitesBackup;
      } catch (error) {
        handleError(error, { context: t('relaySites.buttons.importBackup') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      toast.success(t('relaySites.messages.importBackupSuccess'));
    },
  });
}

export function useSyncRelaySite() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (id: string) => {
      try {
        const data = await graphqlRequest<{ syncRelaySite: RelaySite }>(SYNC_RELAY_SITE_MUTATION, { id });
        return relaySiteSchema.parse(data.syncRelaySite);
      } catch (error) {
        handleError(error, { context: t('relaySites.actions.sync') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      toast.success(t('relaySites.messages.syncSuccess'));
    },
  });
}

export function useSyncAllRelaySites() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async () => {
      try {
        const data = await graphqlRequest<{ syncAllRelaySites: RelaySiteBatchOperationResult }>(SYNC_ALL_RELAY_SITES_MUTATION);
        return relaySiteBatchOperationResultSchema.parse(data.syncAllRelaySites);
      } catch (error) {
        handleError(error, { context: t('relaySites.buttons.syncAll') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
    },
  });
}

export function useCheckinRelaySite() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (id: string) => {
      try {
        const data = await graphqlRequest<{ checkinRelaySite: RelaySiteCheckinLog }>(CHECKIN_RELAY_SITE_MUTATION, { id });
        return relaySiteCheckinLogSchema.parse(data.checkinRelaySite);
      } catch (error) {
        handleError(error, { context: t('relaySites.actions.checkin') });
        throw error;
      }
    },
    onSuccess: log => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      queryClient.invalidateQueries({ queryKey: ['failedRelaySiteCheckinPages'] });
      const message = relaySiteCheckinLogMessage(log);
      if (log.status === 'failed') {
        toast.error(message || t('common.errors.operationFailed', { operation: t('relaySites.actions.checkin') }));
        return;
      }
      if (log.status === 'skipped') {
        toast.info(message || t('relaySites.checkinStatus.skipped'));
        return;
      }
      if (message) {
        toast.success(t('relaySites.messages.checkinSuccess'), { description: message });
        return;
      }
      toast.success(t('relaySites.messages.checkinSuccess'));
    },
  });
}

export function useCheckinAllRelaySites() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async () => {
      try {
        const data = await graphqlRequest<{ checkinAllRelaySites: RelaySiteBatchOperationResult }>(CHECKIN_ALL_RELAY_SITES_MUTATION);
        return relaySiteBatchOperationResultSchema.parse(data.checkinAllRelaySites);
      } catch (error) {
        handleError(error, { context: t('relaySites.buttons.checkinAll') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      queryClient.invalidateQueries({ queryKey: ['failedRelaySiteCheckinPages'] });
    },
  });
}

export function useFailedRelaySiteCheckinPages(enabled = true) {
  const { t } = useTranslation();
  const { handleError } = useErrorHandler();

  return useQuery({
    queryKey: ['failedRelaySiteCheckinPages'],
    queryFn: async () => {
      try {
        const data = await graphqlRequest<{ failedRelaySiteCheckinPages: RelaySiteCheckinPage[] }>(FAILED_RELAY_SITE_CHECKIN_PAGES_QUERY);
        return z.array(relaySiteCheckinPageSchema).parse(data.failedRelaySiteCheckinPages);
      } catch (error) {
        handleError(error, { context: t('relaySites.buttons.openFailedCheckinPages') });
        throw error;
      }
    },
    enabled,
    staleTime: 30_000,
  });
}

function toastBatchOperationResult(result: RelaySiteBatchOperationResult, successMessage: string, partialFailureMessage: string, summary: string, failureSummary: string) {
  if (result.failedCount > 0) {
    toast.warning(partialFailureMessage, { description: failureSummary });
    return;
  }

  toast.success(successMessage, { description: summary });
}

function relaySiteCheckinLogMessage(log: RelaySiteCheckinLog) {
  return (log.errorMessage || log.message || '').trim();
}

async function fetchRelaySiteAnnouncements(id: string) {
  const data = await graphqlRequest<{ node: RelaySiteAnnouncementResult | null }>(RELAY_SITE_ANNOUNCEMENTS_QUERY, { id });
  if (!data.node) {
    throw new Error('relay site not found');
  }
  return relaySiteAnnouncementResultSchema.parse(data.node);
}

export function useRefreshRelaySiteAnnouncements() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (id: string) => {
      try {
        await graphqlRequest<{ refreshRelaySiteAnnouncements: Pick<RelaySite, 'id'> }>(REFRESH_RELAY_SITE_ANNOUNCEMENTS_MUTATION, { id });
        return fetchRelaySiteAnnouncements(id);
      } catch (error) {
        handleError(error, { context: t('relaySites.actions.announcements') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
    },
  });
}

export function useMarkRelaySiteAnnouncementsRead() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (id: string) => {
      try {
        await graphqlRequest<{ markRelaySiteAnnouncementsRead: Pick<RelaySite, 'id'> }>(MARK_RELAY_SITE_ANNOUNCEMENTS_READ_MUTATION, { id });
        return fetchRelaySiteAnnouncements(id);
      } catch (error) {
        handleError(error, { context: t('relaySites.announcements.markRead') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      toast.success(t('relaySites.messages.announcementsRead'));
    },
  });
}

export function useImportRelaySiteAPIKeyToChannel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async ({ relaySiteAPIKeyID, input }: { relaySiteAPIKeyID: string; input: ImportRelaySiteAPIKeyToChannelInput }) => {
      try {
        const validatedInput = importRelaySiteAPIKeyToChannelInputSchema.parse(input);
        const data = await graphqlRequest<{ importRelaySiteAPIKeyToChannel: ImportedRelaySiteChannel }>(
          IMPORT_RELAY_SITE_API_KEY_TO_CHANNEL_MUTATION,
          { relaySiteAPIKeyID, input: validatedInput }
        );
        return importedRelaySiteChannelSchema.parse(data.importRelaySiteAPIKeyToChannel);
      } catch (error) {
        handleError(error, { context: t('relaySites.dialogs.importChannel.title') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      queryClient.invalidateQueries({ queryKey: ['relaySiteChannels'] });
      toast.success(t('relaySites.messages.importChannelSuccess'));
    },
  });
}

export function useRelaySiteChannels(relaySiteID?: string) {
  return useQuery({
    queryKey: ['relaySiteChannels', relaySiteID],
    queryFn: async () => {
      if (!relaySiteID) return [];
      // Extract raw integer ID from GraphQL node ID (gid://axonhub/RelaySite/123 → 123)
      const rawID = relaySiteID.split('/').pop() || '';
      const data = await graphqlRequest<{
        queryChannels: { edges: Array<{ node: { id: string; status: string; tags: string[] | null; endpoints?: Array<{ apiFormat: string; path?: string; baseURL?: string; transport?: string }> | null } }> };
      }>(RELAY_SITE_CHANNELS_QUERY, {
        input: { hasTag: `relay-site:${rawID}`, first: 1000 },
      });
      return data.queryChannels.edges.map((edge) => edge.node);
    },
    enabled: !!relaySiteID,
  });
}

export function useCreateRelaySiteAPIKey() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async ({ relaySiteID, input }: { relaySiteID: string; input: RelaySiteAPIKeyConfigInput }) => {
      try {
        const validatedInput = relaySiteAPIKeyConfigInputSchema.parse(input);
        const data = await graphqlRequest<{ createRelaySiteAPIKey: RelaySite }>(CREATE_RELAY_SITE_API_KEY_MUTATION, {
          relaySiteID,
          input: validatedInput,
        });
        return relaySiteSchema.parse(data.createRelaySiteAPIKey);
      } catch (error) {
        handleError(error, { context: t('relaySites.dialogs.apiKeys.title') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      toast.success(t('relaySites.messages.apiKeySaved'));
    },
  });
}

export function useUpdateRelaySiteAPIKey() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async ({ relaySiteAPIKeyID, input }: { relaySiteAPIKeyID: string; input: RelaySiteAPIKeyConfigInput }) => {
      try {
        const validatedInput = relaySiteAPIKeyConfigInputSchema.parse(input);
        const data = await graphqlRequest<{ updateRelaySiteAPIKey: RelaySite }>(UPDATE_RELAY_SITE_API_KEY_MUTATION, {
          relaySiteAPIKeyID,
          input: validatedInput,
        });
        return relaySiteSchema.parse(data.updateRelaySiteAPIKey);
      } catch (error) {
        handleError(error, { context: t('relaySites.dialogs.apiKeys.title') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      toast.success(t('relaySites.messages.apiKeySaved'));
    },
  });
}

export function useDeleteRelaySiteAPIKey() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (relaySiteAPIKeyID: string) => {
      try {
        const data = await graphqlRequest<{ deleteRelaySiteAPIKey: RelaySite }>(DELETE_RELAY_SITE_API_KEY_MUTATION, { relaySiteAPIKeyID });
        return relaySiteSchema.parse(data.deleteRelaySiteAPIKey);
      } catch (error) {
        handleError(error, { context: t('relaySites.dialogs.apiKeys.title') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      toast.success(t('relaySites.messages.apiKeyDeleted'));
    },
  });
}

export function useCreateRelaySiteAPIKeysForAllGroups() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async ({ relaySiteID, input }: { relaySiteID: string; input: RelaySiteAPIKeyConfigInput }) => {
      try {
        const validatedInput = relaySiteAPIKeyConfigInputSchema.parse(input);
        const data = await graphqlRequest<{ createRelaySiteAPIKeysForAllGroups: RelaySite }>(
          CREATE_RELAY_SITE_API_KEYS_FOR_ALL_GROUPS_MUTATION,
          { relaySiteID, input: validatedInput }
        );
        return relaySiteSchema.parse(data.createRelaySiteAPIKeysForAllGroups);
      } catch (error) {
        handleError(error, { context: t('relaySites.apiKeys.actions.createForAllGroups') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['relaySites'] });
      toast.success(t('relaySites.messages.apiKeysCreatedForAllGroups'));
    },
  });
}
