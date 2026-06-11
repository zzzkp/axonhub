'use client';

import { useMemo, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Boxes, Pencil, Plus, Trash2, UsersRound } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Switch } from '@/components/ui/switch';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { useSaveChannelEndpoints } from '@/features/channels/data/channels';
import type { ChannelEndpoint } from '@/features/channels/data/schema';
import { cn } from '@/lib/utils';
import { useRelaySitesContext } from '../context/relay-sites-context';
import {
  useCreateRelaySiteAPIKey,
  useCreateRelaySiteAPIKeysForAllGroups,
  useDeleteRelaySiteAPIKey,
  useUpdateRelaySiteAPIKey,
  useImportRelaySiteAPIKeyToChannel,
  useRelaySiteChannels,
  useUpdateChannelStatus,
  type RelaySite,
  type RelaySiteAPIKey,
  type RelaySiteAPIKeyConfigInput,
  type RelaySiteModelPrice,
  type ImportRelaySiteAPIKeyToChannelInput,
} from '../data/relay-sites';
import { RelaySiteAPIKeyFormDialog } from './relay-site-api-key-form-dialog';

type APIKeyFormMode = 'create' | 'edit' | 'allGroups';

const MESSAGE_PROTOCOL_ENDPOINTS = [
  'openai/responses',
  'anthropic/messages',
  'gemini/contents',
] as const;

function nodes<T>(connection?: { edges?: Array<{ node: T }> | null } | null) {
  return connection?.edges?.map((edge) => edge.node) ?? [];
}

function formatDate(value?: string | null) {
  if (!value) return '-';
  return format(new Date(value), 'yyyy-MM-dd HH:mm:ss');
}

function formatPrice(value?: string | number | null) {
  if (value == null || value === '') return '-';
  return String(value);
}

function priceText(model: RelaySiteModelPrice, t: (key: string, options?: Record<string, unknown>) => string) {
  if (model.quotaType === 1) {
    return t('relaySites.models.price.perRequest', { price: formatPrice(model.modelPrice ?? model.promptPrice) });
  }
  return t('relaySites.models.price.usage', {
    input: formatPrice(model.modelRatio ?? model.promptPrice),
    output: formatPrice(model.completionRatio ?? model.completionPrice),
  });
}

function billingTypeText(model: RelaySiteModelPrice, t: (key: string) => string) {
  return model.quotaType === 1 ? t('relaySites.models.billingTypes.perRequest') : t('relaySites.models.billingTypes.usage');
}

function modelEnabledForGroup(model: RelaySiteModelPrice, groupName: string) {
  return model.enableGroups.length === 0 || model.enableGroups.includes(groupName);
}

function normalizeBaseURL(value: string) {
  return value.trim().replace(/\/+$/, '');
}

function hasMessageProtocolEndpoints(endpoints: ChannelEndpoint[]) {
  const formats = new Set(endpoints.map((endpoint) => endpoint.apiFormat));
  return MESSAGE_PROTOCOL_ENDPOINTS.every((format) => formats.has(format));
}

// 构建导入 Channel 的默认参数，复用原导入弹窗的默认值逻辑
function buildImportInput(relaySite: RelaySite, apiKey: RelaySiteAPIKey): ImportRelaySiteAPIKeyToChannelInput | null {
  const apiKeyGroupName = apiKey.groupName?.trim() ?? '';
  const models = nodes(relaySite.modelPrices)
    .filter((price) => price.enableGroups.length === 0 || (apiKeyGroupName ? price.enableGroups.includes(apiKeyGroupName) : false))
    .map((price) => price.modelID)
    .sort((left, right) => left.localeCompare(right));

  // 如果该分组无可用模型，返回 null 表示无法导入
  if (models.length === 0) {
    return null;
  }

  return {
    name: `${relaySite.name} - ${apiKey.name || apiKey.remoteID}`,
    type: 'openai',
    baseURL: normalizeBaseURL(relaySite.baseURL),
    supportedModels: models,
    defaultTestModel: models[0],
    tags: ['relay-site'],
    remark: relaySite.remark ?? '',
  };
}

export function RelaySiteModelsAndTokensDialog() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const {
    isModelsAndTokensDialogOpen,
    setIsModelsAndTokensDialogOpen,
    managingRelaySite,
    setManagingRelaySite,
  } = useRelaySitesContext();

  const [selectedGroup, setSelectedGroup] = useState('');
  const [formMode, setFormMode] = useState<APIKeyFormMode>('create');
  const [editingAPIKey, setEditingAPIKey] = useState<RelaySiteAPIKey | null>(null);
  const [isFormDialogOpen, setIsFormDialogOpen] = useState(false);
  const [deletingID, setDeletingID] = useState<string | null>(null);
  const [togglingID, setTogglingID] = useState<string | null>(null);
  const [togglingEndpointsID, setTogglingEndpointsID] = useState<string | null>(null);

  const createMutation = useCreateRelaySiteAPIKey();
  const updateMutation = useUpdateRelaySiteAPIKey();
  const deleteMutation = useDeleteRelaySiteAPIKey();
  const createAllGroupsMutation = useCreateRelaySiteAPIKeysForAllGroups();
  const importMutation = useImportRelaySiteAPIKeyToChannel();
  const updateChannelStatusMutation = useUpdateChannelStatus();
  const saveChannelEndpointsMutation = useSaveChannelEndpoints();
  const pending = createMutation.isPending || updateMutation.isPending || createAllGroupsMutation.isPending;

  const { data: channels = [] } = useRelaySiteChannels(managingRelaySite?.id);

  const groups = useMemo(() => nodes(managingRelaySite?.groups), [managingRelaySite]);
  const apiKeys = useMemo(() => nodes(managingRelaySite?.apiKeys), [managingRelaySite]);
  const modelPrices = useMemo(
    () => [...nodes(managingRelaySite?.modelPrices)].sort((left, right) => left.modelID.localeCompare(right.modelID)),
    [managingRelaySite]
  );

  // 构建 API Key ID → Channel 映射（通过 tags 中的 relay-site-api-key:{id}）
  const apiKeyChannelMap = useMemo(() => {
    const map = new Map<string, { id: string; status: string; endpoints: ChannelEndpoint[] }>();
    channels.forEach((ch) => {
      const tags = ch.tags || [];
      const apiKeyTag = tags.find((tag) => tag.startsWith('relay-site-api-key:'));
      if (apiKeyTag) {
        const apiKeyID = apiKeyTag.split(':')[1];
        // 需要从后端的整数 ID 转换为前端的 GraphQL node ID
        // 但这里 apiKeys 中的 id 已经是 GraphQL ID，我们需要比对
        const apiKey = apiKeys.find((key) => key.id.endsWith(`/${apiKeyID}`));
        if (apiKey) {
          map.set(apiKey.id, { id: ch.id, status: ch.status, endpoints: ch.endpoints ?? [] });
        }
      }
    });
    return map;
  }, [channels, apiKeys]);

  const filteredAPIKeys = useMemo(() => {
    if (selectedGroup === '') return apiKeys;
    return apiKeys.filter((key) => key.groupName === selectedGroup);
  }, [apiKeys, selectedGroup]);

  const visibleModels = useMemo(() => {
    if (selectedGroup === '') return [];
    return modelPrices.filter((model) => modelEnabledForGroup(model, selectedGroup));
  }, [modelPrices, selectedGroup]);

  const setOpen = (open: boolean) => {
    setIsModelsAndTokensDialogOpen(open);
    if (!open) {
      setManagingRelaySite(null);
      setSelectedGroup('');
      setEditingAPIKey(null);
      setDeletingID(null);
      setFormMode('create');
      setTogglingID(null);
      setTogglingEndpointsID(null);
    }
  };

  const handleChannelToggle = async (apiKey: RelaySiteAPIKey, currentlyEnabled: boolean) => {
    if (!managingRelaySite) return;
    setTogglingID(apiKey.id);
    try {
      const channelInfo = apiKeyChannelMap.get(apiKey.id);
      if (currentlyEnabled) {
        // 关闭：禁用对应的 Channel
        if (channelInfo) {
          await updateChannelStatusMutation.mutateAsync({ id: channelInfo.id, status: 'disabled' });
          // 手动刷新 channels 查询，因为 updateChannelStatus 不会自动刷新 relaySiteChannels
          await queryClient.invalidateQueries({ queryKey: ['relaySiteChannels', managingRelaySite.id] });
        }
      } else if (channelInfo) {
        // 已有 Channel（被禁用）：重新启用，避免重复导入
        await updateChannelStatusMutation.mutateAsync({ id: channelInfo.id, status: 'enabled' });
        await queryClient.invalidateQueries({ queryKey: ['relaySiteChannels', managingRelaySite.id] });
      } else {
        // 无 Channel：导入为新的 Channel（启用状态）
        const input = buildImportInput(managingRelaySite, apiKey);
        if (!input) {
          toast.error(t('relaySites.messages.noModelsAvailable'));
          return;
        }
        await importMutation.mutateAsync({ relaySiteAPIKeyID: apiKey.id, input });
        // importMutation 的 onSuccess 已经刷新了 relaySiteChannels
      }
    } catch (error) {
      // 错误已由 mutation 的 handleError 处理
    } finally {
      setTogglingID(null);
    }
  };

  const handleEndpointToggle = async (apiKey: RelaySiteAPIKey, currentlyEnabled: boolean) => {
    if (!managingRelaySite) return;
    const channelInfo = apiKeyChannelMap.get(apiKey.id);
    if (!channelInfo) return;

    setTogglingEndpointsID(apiKey.id);
    try {
      const current = channelInfo.endpoints ?? [];
      const next = currentlyEnabled
        ? current.filter((endpoint) => {
            if (!MESSAGE_PROTOCOL_ENDPOINTS.includes(endpoint.apiFormat as (typeof MESSAGE_PROTOCOL_ENDPOINTS)[number])) return true;
            return Boolean(endpoint.path || endpoint.baseURL || endpoint.transport);
          })
        : [
            ...current,
            ...MESSAGE_PROTOCOL_ENDPOINTS
              .filter((format) => !current.some((endpoint) => endpoint.apiFormat === format))
              .map((format) => ({ apiFormat: format })),
          ];

      await saveChannelEndpointsMutation.mutateAsync({ channelID: channelInfo.id, endpoints: next });
      await queryClient.invalidateQueries({ queryKey: ['relaySiteChannels', managingRelaySite.id] });
    } catch (error) {
      // 错误已由 mutation 的 handleError 处理
    } finally {
      setTogglingEndpointsID(null);
    }
  };

  const startCreate = (mode: APIKeyFormMode) => {
    setFormMode(mode);
    setEditingAPIKey(null);
    setIsFormDialogOpen(true);
  };

  const handleFormSubmit = async (input: RelaySiteAPIKeyConfigInput, mode: APIKeyFormMode, apiKeyID?: string) => {
    if (!managingRelaySite) return;
    if (mode === 'edit' && apiKeyID) {
      const site = await updateMutation.mutateAsync({ relaySiteAPIKeyID: apiKeyID, input });
      setManagingRelaySite(site);
    } else if (mode === 'allGroups') {
      const site = await createAllGroupsMutation.mutateAsync({ relaySiteID: managingRelaySite.id, input });
      setManagingRelaySite(site);
    } else {
      const site = await createMutation.mutateAsync({ relaySiteID: managingRelaySite.id, input });
      setManagingRelaySite(site);
    }
    setEditingAPIKey(null);
    setFormMode('create');
  };

  return (
    <>
      <Dialog open={isModelsAndTokensDialogOpen} onOpenChange={setOpen}>
        <DialogContent className='flex h-[85vh] flex-col sm:max-w-[1400px]'>
          <DialogHeader>
            <DialogTitle>{t('relaySites.dialogs.modelsAndTokens.title')}</DialogTitle>
            <DialogDescription>{managingRelaySite?.name ?? t('relaySites.dialogs.modelsAndTokens.description')}</DialogDescription>
          </DialogHeader>

          <div className='grid min-h-0 flex-1 gap-4 overflow-hidden lg:grid-cols-[200px_minmax(0,1fr)_minmax(0,1fr)]'>
            {/* 左侧：分组列表 */}
            <div className='flex min-h-0 flex-col overflow-hidden rounded-md border'>
              <div className='flex items-center gap-2 border-b px-3 py-2 text-sm font-medium'>
                <Boxes className='h-4 w-4' />
                {t('relaySites.models.groupsTitle')}
              </div>
              <div className='min-h-0 flex-1 space-y-1 overflow-y-auto p-2'>
                <Button
                  type='button'
                  variant='ghost'
                  className={cn(
                    'h-auto w-full flex-col items-start gap-1 px-3 py-2 text-left',
                    selectedGroup === '' && 'bg-muted'
                  )}
                  onClick={() => setSelectedGroup('')}
                >
                  <span className='min-w-0 truncate font-medium'>{t('relaySites.modelsAndTokens.allGroups')}</span>
                  <span className='text-xs text-muted-foreground'>{t('relaySites.modelsAndTokens.groupKeyCount', { count: apiKeys.length })}</span>
                </Button>
                {groups.length > 0 ? (
                  groups.map((group) => {
                    const groupKeyCount = apiKeys.filter((key) => key.groupName === group.name).length;
                    return (
                      <Button
                        key={group.id}
                        type='button'
                        variant='ghost'
                        className={cn(
                          'h-auto w-full flex-col items-start gap-1 px-3 py-2 text-left',
                          selectedGroup === group.name && 'bg-muted'
                        )}
                        onClick={() => setSelectedGroup(group.name)}
                      >
                        <div className='flex w-full items-center justify-between gap-2'>
                          <span className='min-w-0 truncate font-medium'>{group.name}</span>
                          {group.ratio != null && <Badge variant='secondary' className='text-xs'>{group.ratio}</Badge>}
                        </div>
                        <span className='text-xs text-muted-foreground'>{t('relaySites.modelsAndTokens.groupKeyCount', { count: groupKeyCount })}</span>
                      </Button>
                    );
                  })
                ) : (
                  <div className='px-3 py-8 text-center text-sm text-muted-foreground'>{t('relaySites.resources.empty')}</div>
                )}
              </div>
            </div>

            {/* 中间：API Key 列表 */}
            <div className='flex min-h-0 flex-col overflow-hidden rounded-md border'>
              <div className='flex min-h-11 flex-wrap items-center justify-between gap-2 border-b px-3 py-2'>
                <div className='flex items-center gap-2 text-sm font-medium'>
                  {selectedGroup ? t('relaySites.modelsAndTokens.groupTokens', { group: selectedGroup, count: filteredAPIKeys.length }) : t('relaySites.modelsAndTokens.allTokens', { count: apiKeys.length })}
                </div>
                <div className='flex gap-2'>
                  <Button type='button' variant='outline' size='sm' onClick={() => startCreate('create')}>
                    <Plus className='mr-1 h-3.5 w-3.5' />
                    {t('relaySites.apiKeys.actions.create')}
                  </Button>
                  <Button type='button' variant='outline' size='sm' onClick={() => startCreate('allGroups')}>
                    <UsersRound className='mr-1 h-3.5 w-3.5' />
                    {t('relaySites.apiKeys.actions.createForAllGroups')}
                  </Button>
                </div>
              </div>
              <div className='min-h-0 flex-1 overflow-auto'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('relaySites.apiKeys.columns.name')}</TableHead>
                      <TableHead>{t('relaySites.apiKeys.columns.group')}</TableHead>
                      <TableHead>{t('relaySites.modelsAndTokens.channelStatus')}</TableHead>
                      <TableHead>{t('relaySites.modelsAndTokens.endpointConfig')}</TableHead>
                      <TableHead className='w-[100px]'>{t('common.columns.actions')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredAPIKeys.length > 0 ? (
                      filteredAPIKeys.map((apiKey) => {
                        const channelInfo = apiKeyChannelMap.get(apiKey.id);
                        const isEnabled = channelInfo?.status === 'enabled';
                        const isToggling = togglingID === apiKey.id;
                        const hasEndpointConfig = channelInfo ? hasMessageProtocolEndpoints(channelInfo.endpoints) : false;
                        const isEndpointToggling = togglingEndpointsID === apiKey.id;
                        return (
                          <TableRow key={apiKey.id}>
                            <TableCell className='min-w-[160px]'>
                              <div className='font-medium'>{apiKey.name || apiKey.remoteID}</div>
                              <div className='font-mono text-xs text-muted-foreground'>{apiKey.remoteID}</div>
                            </TableCell>
                            <TableCell>{apiKey.groupName || '-'}</TableCell>
                            <TableCell>
                              <Switch
                                checked={isEnabled}
                                disabled={isToggling}
                                onCheckedChange={() => handleChannelToggle(apiKey, isEnabled)}
                              />
                            </TableCell>
                            <TableCell>
                              <Switch
                                checked={hasEndpointConfig}
                                disabled={!channelInfo || isEndpointToggling || saveChannelEndpointsMutation.isPending}
                                onCheckedChange={() => handleEndpointToggle(apiKey, hasEndpointConfig)}
                              />
                            </TableCell>
                            <TableCell>
                              <div className='flex items-center gap-1'>
                                <Button
                                  type='button'
                                  variant='ghost'
                                  size='icon'
                                  className='h-8 w-8'
                                  onClick={() => {
                                    setEditingAPIKey(apiKey);
                                    setFormMode('edit');
                                    setIsFormDialogOpen(true);
                                  }}
                                >
                                  <span className='sr-only'>{t('common.buttons.edit')}</span>
                                  <Pencil className='h-4 w-4' />
                                </Button>
                                <Button
                                  type='button'
                                  variant={deletingID === apiKey.id ? 'destructive' : 'ghost'}
                                  size='icon'
                                  className='h-8 w-8'
                                  disabled={deleteMutation.isPending}
                                  onClick={async () => {
                                    if (deletingID !== apiKey.id) {
                                      setDeletingID(apiKey.id);
                                      return;
                                    }
                                    const site = await deleteMutation.mutateAsync(apiKey.id);
                                    setManagingRelaySite(site);
                                    setDeletingID(null);
                                  }}
                                >
                                  <span className='sr-only'>{t('common.buttons.delete')}</span>
                                  <Trash2 className='h-4 w-4' />
                                </Button>
                              </div>
                            </TableCell>
                          </TableRow>
                        );
                      })
                    ) : (
                      <TableRow>
                        <TableCell colSpan={5} className='h-24 text-center'>
                          {t('common.noData')}
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>

            {/* 右侧：模型价格列表 */}
            <div className='flex min-h-0 flex-col overflow-hidden rounded-md border'>
              <div className='flex min-h-11 flex-wrap items-center justify-between gap-2 border-b px-3 py-2'>
                <div className='min-w-0 text-sm font-medium'>
                  {selectedGroup
                    ? t('relaySites.models.groupModels', { group: selectedGroup, count: visibleModels.length })
                    : t('relaySites.models.noGroupSelected')}
                </div>
                <div className='text-xs text-muted-foreground'>{t('relaySites.models.total', { count: modelPrices.length })}</div>
              </div>
              <div className='min-h-0 flex-1 overflow-auto'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('relaySites.models.columns.model')}</TableHead>
                      <TableHead>{t('relaySites.models.columns.billingType')}</TableHead>
                      <TableHead>{t('relaySites.models.columns.price')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {visibleModels.length > 0 ? (
                      visibleModels.map((model) => (
                        <TableRow key={model.id}>
                          <TableCell className='min-w-[180px] font-mono text-sm'>{model.modelID}</TableCell>
                          <TableCell className='whitespace-nowrap text-sm'>
                            <Badge variant='secondary'>{billingTypeText(model, t)}</Badge>
                          </TableCell>
                          <TableCell className='min-w-[160px] whitespace-nowrap font-mono text-sm'>{priceText(model, t)}</TableCell>
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={3} className='h-24 text-center'>
                          {t('common.noData')}
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <RelaySiteAPIKeyFormDialog
        open={isFormDialogOpen}
        onOpenChange={setIsFormDialogOpen}
        relaySite={managingRelaySite}
        editingAPIKey={editingAPIKey}
        mode={formMode}
        onSubmit={handleFormSubmit}
        pending={pending}
      />
    </>
  );
}
