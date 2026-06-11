'use client';

import { useEffect } from 'react';
import { KeyRound, UsersRound } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import type { RelaySite, RelaySiteAPIKey, RelaySiteAPIKeyConfigInput } from '../data/relay-sites';

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

interface APIKeyFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  relaySite: RelaySite | null;
  editingAPIKey: RelaySiteAPIKey | null;
  mode: APIKeyFormMode;
  onSubmit: (input: RelaySiteAPIKeyConfigInput, mode: APIKeyFormMode, apiKeyID?: string) => Promise<void>;
  pending: boolean;
}

export function RelaySiteAPIKeyFormDialog({
  open,
  onOpenChange,
  relaySite,
  editingAPIKey,
  mode,
  onSubmit,
  pending,
}: APIKeyFormDialogProps) {
  const { t } = useTranslation();
  const groups = nodes(relaySite?.groups);

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<APIKeyFormValues>({ defaultValues: defaultValues(null) });

  useEffect(() => {
    if (!open) return;
    reset(defaultValues(relaySite, editingAPIKey));
  }, [editingAPIKey, open, relaySite, reset]);

  const handleFormSubmit = async (data: APIKeyFormValues) => {
    const input = toInput(data);
    await onSubmit(input, mode, editingAPIKey?.id);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-[480px]'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            {mode === 'allGroups' ? <UsersRound className='h-5 w-5' /> : <KeyRound className='h-5 w-5' />}
            {t(`relaySites.apiKeys.form.${mode}`)}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(handleFormSubmit)} className='space-y-4' noValidate>
          <div className='grid gap-2'>
            <Label htmlFor='api-key-name'>{t('relaySites.apiKeys.fields.name')}</Label>
            <Input id='api-key-name' {...register('name', { required: t('relaySites.validation.apiKeyNameRequired') })} />
            {errors.name && <span className='text-sm text-red-500'>{errors.name.message}</span>}
          </div>

          <div className='grid grid-cols-2 gap-3'>
            <div className='grid gap-2'>
              <Label htmlFor='api-key-status'>{t('common.columns.status')}</Label>
              <Select value={watch('status')} onValueChange={(value) => setValue('status', value)}>
                <SelectTrigger id='api-key-status'><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value='1'>{t('relaySites.apiKeys.status.enabled')}</SelectItem>
                  <SelectItem value='2'>{t('relaySites.apiKeys.status.disabled')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='api-key-group'>{t('relaySites.apiKeys.fields.group')}</Label>
              <Select value={watch('group') || '__none'} onValueChange={(value) => setValue('group', value === '__none' ? '' : value)} disabled={mode === 'allGroups'}>
                <SelectTrigger id='api-key-group'><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value='__none'>{t('relaySites.apiKeys.fields.noGroup')}</SelectItem>
                  {groups.map((group) => <SelectItem key={group.id} value={group.name}>{group.name}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className='grid grid-cols-2 gap-3'>
            <div className='grid gap-2'>
              <Label htmlFor='api-key-quota'>{t('relaySites.apiKeys.fields.remainQuota')}</Label>
              <Input id='api-key-quota' type='number' min='0' step='1' disabled={watch('unlimitedQuota')} {...register('remainQuota')} />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='api-key-expires-at'>{t('relaySites.apiKeys.fields.expiresAt')}</Label>
              <Input id='api-key-expires-at' type='datetime-local' {...register('expiresAt')} />
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
            <Label htmlFor='api-key-model-limits'>{t('relaySites.apiKeys.fields.modelLimits')}</Label>
            <Textarea id='api-key-model-limits' rows={2} disabled={!watch('modelLimitsEnabled')} {...register('modelLimits')} />
          </div>

          <div className='grid gap-2'>
            <Label htmlFor='api-key-allow-ips'>{t('relaySites.apiKeys.fields.allowIps')}</Label>
            <Input id='api-key-allow-ips' {...register('allowIps')} />
          </div>

          <DialogFooter>
            <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
              {t('common.buttons.cancel')}
            </Button>
            <Button type='submit' disabled={pending}>
              {pending ? t('common.buttons.saving') : t('common.buttons.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
