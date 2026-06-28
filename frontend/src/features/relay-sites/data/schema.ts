import { z } from 'zod';
import { pageInfoSchema } from '@/gql/pagination';
import { channelTypeSchema } from '@/features/channels/data/schema';

export const relaySiteStatusSchema = z.enum(['enabled', 'disabled', 'archived']);
export const relaySiteTypeSchema = z.enum(['new_api', 'sub2api', 'done_hub']);
export const relaySiteNatureSchema = z.enum(['public', 'semi_public', 'paid']);
export const relaySiteCredentialAuthTypeSchema = z.enum(['token', 'password', 'jwt']);

export const relaySiteAPIKeySchema = z.object({
  id: z.string(),
  remoteID: z.string(),
  name: z.string().nullable().optional(),
  status: z.enum(['enabled', 'disabled', 'unknown']),
  groupName: z.string().nullable().optional(),
  quota: z.number().nullable().optional(),
  usedQuota: z.number().nullable().optional(),
  expiresAt: z.string().nullable().optional(),
  syncedAt: z.string(),
});
export type RelaySiteAPIKey = z.infer<typeof relaySiteAPIKeySchema>;

export const relaySiteGroupSchema = z.object({
  id: z.string(),
  name: z.string(),
  ratio: z.number().nullable().optional(),
  syncedAt: z.string(),
});
export type RelaySiteGroup = z.infer<typeof relaySiteGroupSchema>;

export const relaySiteBalanceSnapshotSchema = z.object({
  id: z.string(),
  balance: z.number(),
  unit: z.string(),
  pulledAt: z.string(),
});
export type RelaySiteBalanceSnapshot = z.infer<typeof relaySiteBalanceSnapshotSchema>;

export const relaySiteDisplayCredentialSchema = z.object({
  authType: relaySiteCredentialAuthTypeSchema,
  token: z.string().nullable().optional(),
  userId: z.number().nullable().optional(),
  username: z.string().nullable().optional(),
  password: z.string().nullable().optional(),
  refreshToken: z.string().nullable().optional(),
  tokenExpiresAt: z.string().nullable().optional(),
}).nullable().optional();
export type RelaySiteDisplayCredential = z.infer<typeof relaySiteDisplayCredentialSchema>;

export const relaySiteModelPriceSchema = z.object({
  id: z.string(),
  modelID: z.string(),
  promptPrice: z.union([z.number(), z.string()]).nullable().optional(),
  completionPrice: z.union([z.number(), z.string()]).nullable().optional(),
  billingUnit: z.string().nullable().optional(),
  enableGroups: z.array(z.string()).default([]),
  quotaType: z.number().nullable().optional(),
  modelPrice: z.union([z.number(), z.string()]).nullable().optional(),
  modelRatio: z.union([z.number(), z.string()]).nullable().optional(),
  completionRatio: z.union([z.number(), z.string()]).nullable().optional(),
  supportedEndpointTypes: z.array(z.string()).default([]),
  syncedAt: z.string(),
});
export type RelaySiteModelPrice = z.infer<typeof relaySiteModelPriceSchema>;

export const relaySiteCheckinLogSchema = z.object({
  id: z.string(),
  executedAt: z.string(),
  status: z.enum(['success', 'failed', 'skipped']),
  message: z.string().nullable().optional(),
  errorMessage: z.string().nullable().optional(),
});
export type RelaySiteCheckinLog = z.infer<typeof relaySiteCheckinLogSchema>;

export const relaySiteAnnouncementSchema = z.object({
  id: z.string(),
  remoteID: z.string(),
  type: z.string().nullable().optional(),
  content: z.string(),
  extra: z.string().nullable().optional(),
  publishedAt: z.string().nullable().optional(),
  fetchedAt: z.string(),
  readAt: z.string().nullable().optional(),
});
export type RelaySiteAnnouncement = z.infer<typeof relaySiteAnnouncementSchema>;

const edgeOf = <T extends z.ZodTypeAny>(schema: T) => z.object({ node: schema });

const nestedConnectionOf = <T extends z.ZodTypeAny>(schema: T) =>
  z.object({
    edges: z.array(edgeOf(schema)).nullable().optional(),
    totalCount: z.number(),
  });

export const relaySiteSchema = z.object({
  id: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
  name: z.string(),
  nature: relaySiteNatureSchema,
  type: relaySiteTypeSchema,
  baseURL: z.string(),
  status: relaySiteStatusSchema,
  autoCheckinEnabled: z.boolean(),
  remark: z.string().nullable().optional(),
  lastSyncedAt: z.string().nullable().optional(),
  lastSyncError: z.string().nullable().optional(),
  lastCheckinAt: z.string().nullable().optional(),
  lastCheckinResult: z.string().nullable().optional(),
  checkinPageURL: z.string().nullable().optional(),
  externalCheckinPageURL: z.string().nullable().optional(),
  hasUnreadAnnouncements: z.boolean(),
  displayCredential: relaySiteDisplayCredentialSchema,
  apiKeys: nestedConnectionOf(relaySiteAPIKeySchema),
  groups: nestedConnectionOf(relaySiteGroupSchema),
  balanceSnapshots: nestedConnectionOf(relaySiteBalanceSnapshotSchema),
  modelPrices: nestedConnectionOf(relaySiteModelPriceSchema),
  checkinLogs: nestedConnectionOf(relaySiteCheckinLogSchema),
  announcements: nestedConnectionOf(relaySiteAnnouncementSchema),
});
export type RelaySite = z.infer<typeof relaySiteSchema>;

export const relaySiteFormResultSchema = relaySiteSchema.pick({
  id: true,
  createdAt: true,
  updatedAt: true,
  name: true,
  nature: true,
  type: true,
  baseURL: true,
  status: true,
  autoCheckinEnabled: true,
  remark: true,
  lastSyncedAt: true,
  lastSyncError: true,
  lastCheckinAt: true,
  lastCheckinResult: true,
  checkinPageURL: true,
  externalCheckinPageURL: true,
  hasUnreadAnnouncements: true,
});
export type RelaySiteFormResult = z.infer<typeof relaySiteFormResultSchema>;

export const relaySiteAnnouncementResultSchema = relaySiteSchema.pick({
  id: true,
  name: true,
  hasUnreadAnnouncements: true,
  announcements: true,
});
export type RelaySiteAnnouncementResult = z.infer<typeof relaySiteAnnouncementResultSchema>;

export const relaySitesConnectionSchema = z.object({
  edges: z.array(z.object({ node: relaySiteSchema })),
  pageInfo: pageInfoSchema,
  totalCount: z.number(),
});
export type RelaySitesConnection = z.infer<typeof relaySitesConnectionSchema>;

export const relaySiteBatchOperationFailureSchema = z.object({
  relaySiteID: z.string(),
  relaySiteName: z.string(),
  errorMessage: z.string(),
});
export type RelaySiteBatchOperationFailure = z.infer<typeof relaySiteBatchOperationFailureSchema>;

export const relaySiteBatchOperationResultSchema = z.object({
  totalCount: z.number(),
  successCount: z.number(),
  failedCount: z.number(),
  failures: z.array(relaySiteBatchOperationFailureSchema),
});
export type RelaySiteBatchOperationResult = z.infer<typeof relaySiteBatchOperationResultSchema>;

export const relaySiteCheckinPageSchema = z.object({
  relaySiteID: z.string(),
  relaySiteName: z.string(),
  url: z.string(),
});
export type RelaySiteCheckinPage = z.infer<typeof relaySiteCheckinPageSchema>;

export const relaySiteCredentialInputSchema = z.object({
  authType: relaySiteCredentialAuthTypeSchema,
  token: z.string().optional(),
  userId: z.number().optional(),
  username: z.string().optional(),
  password: z.string().optional(),
  refreshToken: z.string().optional(),
  tokenExpiresAt: z.string().optional(),
});
export type RelaySiteCredentialInput = z.infer<typeof relaySiteCredentialInputSchema>;

export const importedRelaySiteChannelSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: channelTypeSchema,
  baseURL: z.string().nullable().optional(),
  status: z.string(),
  supportedModels: z.array(z.string()),
  defaultTestModel: z.string(),
});
export type ImportedRelaySiteChannel = z.infer<typeof importedRelaySiteChannelSchema>;

export const createRelaySiteInputSchema = z.object({
  name: z.string().min(1),
  nature: relaySiteNatureSchema.optional(),
  type: relaySiteTypeSchema.optional(),
  baseURL: z.string().min(1),
  status: relaySiteStatusSchema.optional(),
  autoCheckinEnabled: z.boolean().optional(),
  remark: z.string().optional(),
  checkinPageURL: z.string().optional(),
  externalCheckinPageURL: z.string().optional(),
  credential: relaySiteCredentialInputSchema,
});
export type CreateRelaySiteInput = z.infer<typeof createRelaySiteInputSchema>;

export const updateRelaySiteInputSchema = z.object({
  name: z.string().min(1).optional(),
  nature: relaySiteNatureSchema.optional(),
  baseURL: z.string().min(1).optional(),
  status: relaySiteStatusSchema.optional(),
  autoCheckinEnabled: z.boolean().optional(),
  remark: z.string().optional(),
  checkinPageURL: z.string().optional(),
  externalCheckinPageURL: z.string().optional(),
  credential: relaySiteCredentialInputSchema.optional(),
});
export type UpdateRelaySiteInput = z.infer<typeof updateRelaySiteInputSchema>;

export const importRelaySiteAPIKeyToChannelInputSchema = z.object({
  name: z.string().min(1),
  type: channelTypeSchema,
  baseURL: z.string().min(1),
  supportedModels: z.array(z.string()).min(1),
  defaultTestModel: z.string().min(1),
  tags: z.array(z.string()).optional(),
  remark: z.string().optional(),
});
export type ImportRelaySiteAPIKeyToChannelInput = z.infer<typeof importRelaySiteAPIKeyToChannelInputSchema>;

export const importRelaySitesBackupInputSchema = z.object({
  payload: z.string().min(1),
});
export type ImportRelaySitesBackupInput = z.infer<typeof importRelaySitesBackupInputSchema>;

export const relaySiteAPIKeyConfigInputSchema = z.object({
  name: z.string().min(1),
  status: z.number().optional(),
  expiredTime: z.number().optional(),
  remainQuota: z.number().optional(),
  unlimitedQuota: z.boolean().optional(),
  modelLimitsEnabled: z.boolean().optional(),
  modelLimits: z.string().optional(),
  allowIps: z.string().optional(),
  group: z.string().optional(),
  crossGroupRetry: z.boolean().optional(),
});
export type RelaySiteAPIKeyConfigInput = z.infer<typeof relaySiteAPIKeyConfigInputSchema>;
