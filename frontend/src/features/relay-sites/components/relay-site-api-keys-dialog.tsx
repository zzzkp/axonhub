'use client';

import { useEffect, useMemo, useState } from 'react';
import { format } from 'date-fns';
import { KeyRound, Pencil, Plus, Trash2, Upload, UsersRound } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { useRelaySitesContext } from '../context/relay-sites-context';
import {
  useCreateRelaySiteAPIKey,
  useCreateRelaySiteAPIKeysForAllGroups,
  useDeleteRelaySiteAPIKey,
  useUpdateRelaySiteAPIKey,
  type RelaySite,
  type RelaySiteAPIKey,
  type RelaySiteAPIKeyConfigInput,
} from '../data/relay-sites';

type APIKeyFormMode = 'create' | 'edit' | 'allGroups';

type APIKeyFormValues = {
  name: string;
  status: string;
  group: string;
  remainQuota: string;
  unlimitedQuota: boolean;
  expiresAt: string;
  modelLimitsEnabled: boolean;
  modelLimits: string;
  allowIps: string;
  crossGroupRetry: boolean;
};

function nodes<T>(connection?: { edges?: Array<{ node: T }> | null } | null) {
  return connection?.edges?.map((edge) => edge.node) ?? [];
}

function formatDate(value?: string | null) {
  if (!value) return '-';
  return format(new Date(value), 'yyyy-MM-dd HH:mm:ss');
}

function toDatetimeLocal(value?: string | null) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function toUnixSeconds(value: string) {
  if (!value) return -1;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return -1;
  return Math.floor(date.getTime() / 1000);
}

function numberOrUndefined(value: string) {
  if (value.trim() === '') return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function defaultValues(relaySite: RelaySite | null, apiKey?: RelaySiteAPIKey | null): APIKeyFormValues {
  const firstGroup = nodes(relaySite?.groups)[0]?.name ?? '';
  return {
    name: apiKey?.name ?? '',
    status: apiKey?.status === 'disabled' ? '2' : '1',
    group: apiKey?.groupName ?? firstGroup,
    remainQuota: apiKey?.quota != null ? String(apiKey.quota) : '',
    unlimitedQuota: apiKey?.quota == null,
    expiresAt: toDatetimeLocal(apiKey?.expiresAt),
    modelLimitsEnabled: false,
    modelLimits: '',
    allowIps: '',
    crossGroupRetry: false,
  };
}

function toInput(data: APIKeyFormValues): RelaySiteAPIKeyConfigInput {
  const input: RelaySiteAPIKeyConfigInput = {
    name: data.name.trim(),
    status: Number(data.status),
    expiredTime: toUnixSeconds(data.expiresAt),
    unlimitedQuota: data.unlimitedQuota,
    modelLimitsEnabled: data.modelLimitsEnabled,
    crossGroupRetry: data.crossGroupRetry,
  };
  const remainQuota = numberOrUndefined(data.remainQuota);
  if (remainQuota != null) input.remainQuota = remainQuota;
  if (data.group.trim()) input.group = data.group.trim();
  if (data.modelLimits.trim()) input.modelLimits = data.modelLimits.trim();
  if (data.allowIps.trim()) input.allowIps = data.allowIps.trim();
  return input;
}

export function RelaySiteAPIKeysDialog() {
  const { t } = useTranslation();
  const {
    isAPIKeysDialogOpen,
    setIsAPIKeysDialogOpen,
    managingRelaySite,
    setManagingRelaySite,
    setImportingRelaySite,
    setImportingAPIKey,
    setIsImportChannelDialogOpen,
  } = useRelaySitesContext();
  const [formMode, setFormMode] = useState<APIKeyFormMode>('create');
  const [editingAPIKey, setEditingAPIKey] = useState<RelaySiteAPIKey | null>(null);
  const [deletingID, setDeletingID] = useState<string | null>(null);

  const createMutation = useCreateRelaySiteAPIKey();
  const updateMutation = useUpdateRelaySiteAPIKey();
  const deleteMutation = useDeleteRelaySiteAPIKey();
  const createAllGroupsMutation = useCreateRelaySiteAPIKeysForAllGroups();
  const pending = createMutation.isPending || updateMutation.isPending || createAllGroupsMutation.isPending;

  const apiKeys = useMemo(() => nodes(managingRelaySite?.apiKeys), [managingRelaySite]);
  const groups = useMemo(() => nodes(managingRelaySite?.groups), [managingRelaySite]);

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<APIKeyFormValues>({ defaultValues: defaultValues(null) });

  useEffect(() => {
    if (!isAPIKeysDialogOpen) return;
    reset(defaultValues(managingRelaySite, editingAPIKey));
  }, [editingAPIKey, isAPIKeysDialogOpen, managingRelaySite, reset]);

  const setOpen = (open: boolean) => {
    setIsAPIKeysDialogOpen(open);
    if (!open) {
      setManagingRelaySite(null);
      setEditingAPIKey(null);
      setDeletingID(null);
      setFormMode('create');
    }
  };

  const startCreate = (mode: APIKeyFormMode) => {
    setFormMode(mode);
    setEditingAPIKey(null);
    reset({ ...defaultValues(managingRelaySite), name: mode === 'allGroups' ? 'group-key' : '' });
  };

  const onSubmit = async (data: APIKeyFormValues) => {
    if (!managingRelaySite) return;
    const input = toInput(data);
    if (formMode === 'edit' && editingAPIKey) {
      const site = await updateMutation.mutateAsync({ relaySiteAPIKeyID: editingAPIKey.id, input });
      setManagingRelaySite(site);
    } else if (formMode === 'allGroups') {
      const site = await createAllGroupsMutation.mutateAsync({ relaySiteID: managingRelaySite.id, input });
      setManagingRelaySite(site);
    } else {
      const site = await createMutation.mutateAsync({ relaySiteID: managingRelaySite.id, input });
      setManagingRelaySite(site);
    }
    setEditingAPIKey(null);
    setFormMode('create');
    reset(defaultValues(managingRelaySite));
  };

  return (
    <Dialog open={isAPIKeysDialogOpen} onOpenChange={setOpen}>
      <DialogContent className='sm:max-w-[1120px]'>
        <DialogHeader>
          <DialogTitle>{t('relaySites.dialogs.apiKeys.title')}</DialogTitle>
          <DialogDescription>{managingRelaySite?.name ?? t('relaySites.dialogs.apiKeys.description')}</DialogDescription>
        </DialogHeader>

        <div className='grid max-h-[72vh] gap-4 overflow-y-auto lg:grid-cols-[minmax(0,1fr)_360px]'>
          <div className='overflow-hidden rounded-md border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('relaySites.apiKeys.columns.name')}</TableHead>
                  <TableHead>{t('relaySites.apiKeys.columns.group')}</TableHead>
                  <TableHead>{t('common.columns.status')}</TableHead>
                  <TableHead>{t('relaySites.apiKeys.columns.quota')}</TableHead>
                  <TableHead>{t('relaySites.apiKeys.columns.expiresAt')}</TableHead>
                  <TableHead className='w-[170px]'>{t('common.columns.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apiKeys.length > 0 ? apiKeys.map((apiKey) => (
                  <TableRow key={apiKey.id}>
                    <TableCell className='min-w-[160px]'>
                      <div className='font-medium'>{apiKey.name || apiKey.remoteID}</div>
                      <div className='font-mono text-xs text-muted-foreground'>{apiKey.remoteID}</div>
                    </TableCell>
                    <TableCell>{apiKey.groupName || '-'}</TableCell>
                    <TableCell><Badge variant={apiKey.status === 'enabled' ? 'default' : 'secondary'}>{apiKey.status}</Badge></TableCell>
                    <TableCell className='whitespace-nowrap text-sm text-muted-foreground'>
                      {apiKey.quota ?? '-'} / {apiKey.usedQuota ?? '-'}
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-sm text-muted-foreground'>{formatDate(apiKey.expiresAt)}</TableCell>
                    <TableCell>
                      <div className='flex items-center gap-1'>
                        <Button type='button' variant='ghost' size='icon' className='h-8 w-8' onClick={() => {
                          setEditingAPIKey(apiKey);
                          setFormMode('edit');
                        }}>
                          <span className='sr-only'>{t('common.buttons.edit')}</span>
                          <Pencil className='h-4 w-4' />
                        </Button>
                        <Button type='button' variant='ghost' size='icon' className='h-8 w-8' onClick={() => {
                          if (!managingRelaySite) return;
                          setImportingRelaySite(managingRelaySite);
                          setImportingAPIKey(apiKey);
                          setIsAPIKeysDialogOpen(false);
                          setIsImportChannelDialogOpen(true);
                        }}>
                          <span className='sr-only'>{t('relaySites.actions.importChannel')}</span>
                          <Upload className='h-4 w-4' />
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
                )) : (
                  <TableRow>
                    <TableCell colSpan={6} className='h-24 text-center'>{t('common.noData')}</TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>

          <form onSubmit={handleSubmit(onSubmit, () => {})} className='space-y-4' noValidate>
            <div className='flex flex-wrap gap-2'>
              <Button type='button' variant={formMode === 'create' ? 'default' : 'outline'} size='sm' onClick={() => startCreate('create')}>
                <Plus className='mr-2 h-4 w-4' />
                {t('relaySites.apiKeys.actions.create')}
              </Button>
              <Button type='button' variant={formMode === 'allGroups' ? 'default' : 'outline'} size='sm' onClick={() => startCreate('allGroups')}>
                <UsersRound className='mr-2 h-4 w-4' />
                {t('relaySites.apiKeys.actions.createForAllGroups')}
              </Button>
            </div>

            <div className='grid gap-3 rounded-md border p-4'>
              <div className='flex items-center gap-2 text-sm font-medium'>
                <KeyRound className='h-4 w-4' />
                {t(`relaySites.apiKeys.form.${formMode}`)}
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='relay-site-api-key-name'>{t('relaySites.apiKeys.fields.name')}</Label>
                <Input id='relay-site-api-key-name' {...register('name', { required: t('relaySites.validation.apiKeyNameRequired') })} />
                {errors.name && <span className='text-sm text-red-500'>{errors.name.message}</span>}
              </div>
              <div className='grid grid-cols-2 gap-3'>
                <div className='grid gap-2'>
                  <Label htmlFor='relay-site-api-key-status'>{t('common.columns.status')}</Label>
                  <Select value={watch('status')} onValueChange={(value) => setValue('status', value)}>
                    <SelectTrigger id='relay-site-api-key-status'><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value='1'>{t('relaySites.apiKeys.status.enabled')}</SelectItem>
                      <SelectItem value='2'>{t('relaySites.apiKeys.status.disabled')}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor='relay-site-api-key-group'>{t('relaySites.apiKeys.fields.group')}</Label>
                  <Select value={watch('group') || '__none'} onValueChange={(value) => setValue('group', value === '__none' ? '' : value)} disabled={formMode === 'allGroups'}>
                    <SelectTrigger id='relay-site-api-key-group'><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value='__none'>{t('relaySites.apiKeys.fields.noGroup')}</SelectItem>
                      {groups.map((group) => <SelectItem key={group.id} value={group.name}>{group.name}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className='grid grid-cols-2 gap-3'>
                <div className='grid gap-2'>
                  <Label htmlFor='relay-site-api-key-quota'>{t('relaySites.apiKeys.fields.remainQuota')}</Label>
                  <Input id='relay-site-api-key-quota' type='number' min='0' step='1' disabled={watch('unlimitedQuota')} {...register('remainQuota')} />
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor='relay-site-api-key-expires-at'>{t('relaySites.apiKeys.fields.expiresAt')}</Label>
                  <Input id='relay-site-api-key-expires-at' type='datetime-local' {...register('expiresAt')} />
                </div>
              </div>
              <div className='flex flex-wrap gap-4'>
                <label className='flex items-center gap-2 text-sm'>
                  <Switch checked={watch('unlimitedQuota')} onCheckedChange={(checked) => setValue('unlimitedQuota', checked)} />
                  {t('relaySites.apiKeys.fields.unlimitedQuota')}
                </label>
                <label className='flex items-center gap-2 text-sm'>
                  <Switch checked={watch('modelLimitsEnabled')} onCheckedChange={(checked) => setValue('modelLimitsEnabled', checked)} />
                  {t('relaySites.apiKeys.fields.modelLimitsEnabled')}
                </label>
                <label className='flex items-center gap-2 text-sm'>
                  <Switch checked={watch('crossGroupRetry')} onCheckedChange={(checked) => setValue('crossGroupRetry', checked)} />
                  {t('relaySites.apiKeys.fields.crossGroupRetry')}
                </label>
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='relay-site-api-key-model-limits'>{t('relaySites.apiKeys.fields.modelLimits')}</Label>
                <Textarea id='relay-site-api-key-model-limits' rows={2} disabled={!watch('modelLimitsEnabled')} {...register('modelLimits')} />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='relay-site-api-key-allow-ips'>{t('relaySites.apiKeys.fields.allowIps')}</Label>
                <Input id='relay-site-api-key-allow-ips' {...register('allowIps')} />
              </div>
            </div>

            <DialogFooter>
              <Button type='submit' disabled={pending || !managingRelaySite}>
                {pending ? t('common.buttons.saving') : t('common.buttons.save')}
              </Button>
            </DialogFooter>
          </form>
        </div>
      </DialogContent>
    </Dialog>
  );
}
