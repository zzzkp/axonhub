'use client';

import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { CHANNEL_CONFIGS } from '@/features/channels/data/config_channels';
import type { ChannelType } from '@/features/channels/data/schema';
import { useRelaySitesContext } from '../context/relay-sites-context';
import { useImportRelaySiteAPIKeyToChannel, type ImportRelaySiteAPIKeyToChannelInput } from '../data/relay-sites';

type ImportChannelFormValues = {
  name: string;
  type: ChannelType;
  baseURL: string;
  supportedModels: string;
  defaultTestModel: string;
  tags: string;
  remark: string;
};

function connectionNodes<T>(connection?: { edges?: Array<{ node: T }> | null } | null) {
  return connection?.edges?.map((edge) => edge.node) ?? [];
}

function parseList(value: string) {
  return Array.from(
    new Set(
      value
        .split(/[\n,]/)
        .map((item) => item.trim())
        .filter(Boolean)
    )
  );
}

function normalizeBaseURL(value: string) {
  return value.trim().replace(/\/+$/, '');
}

function modelEnabledForGroup(model: { enableGroups: string[] }, groupName?: string | null) {
  if (model.enableGroups.length === 0) return true;
  if (!groupName) return false;
  return model.enableGroups.includes(groupName);
}

export function ImportRelaySiteChannelDialog() {
  const { t } = useTranslation();
  const {
    isImportChannelDialogOpen,
    setIsImportChannelDialogOpen,
    importingRelaySite,
    setImportingRelaySite,
    importingAPIKey,
    setImportingAPIKey,
  } = useRelaySitesContext();
  const importMutation = useImportRelaySiteAPIKeyToChannel();

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<ImportChannelFormValues>({
    defaultValues: {
      name: '',
      type: 'openai',
      baseURL: '',
      supportedModels: '',
      defaultTestModel: '',
      tags: '',
      remark: '',
    },
  });

  useEffect(() => {
    if (!isImportChannelDialogOpen || !importingRelaySite || !importingAPIKey) return;

    const apiKeyGroupName = importingAPIKey.groupName?.trim() ?? '';
    const models = connectionNodes(importingRelaySite.modelPrices)
      .filter((price) => modelEnabledForGroup(price, apiKeyGroupName))
      .map((price) => price.modelID)
      .sort((left, right) => left.localeCompare(right));
    const defaultModel = models[0] ?? '';
    reset({
      name: `${importingRelaySite.name} - ${importingAPIKey.name || importingAPIKey.remoteID}`,
      type: 'openai',
      baseURL: normalizeBaseURL(importingRelaySite.baseURL),
      supportedModels: models.join('\n'),
      defaultTestModel: defaultModel,
      tags: 'relay-site',
      remark: importingRelaySite.remark ?? '',
    });
  }, [importingAPIKey, importingRelaySite, isImportChannelDialogOpen, reset]);

  const setOpen = (open: boolean) => {
    setIsImportChannelDialogOpen(open);
    if (!open) {
      setImportingRelaySite(null);
      setImportingAPIKey(null);
    }
  };

  const onSubmit = async (data: ImportChannelFormValues) => {
    if (!importingAPIKey) return;

    const supportedModels = parseList(data.supportedModels);
    const input: ImportRelaySiteAPIKeyToChannelInput = {
      name: data.name.trim(),
      type: data.type,
      baseURL: normalizeBaseURL(data.baseURL),
      supportedModels,
      defaultTestModel: data.defaultTestModel.trim(),
      tags: parseList(data.tags),
      remark: data.remark.trim(),
    };

    await importMutation.mutateAsync({ relaySiteAPIKeyID: importingAPIKey.id, input });
    setOpen(false);
  };

  const channelTypes = Object.values(CHANNEL_CONFIGS);

  return (
    <Dialog open={isImportChannelDialogOpen} onOpenChange={setOpen}>
      <DialogContent className='sm:max-w-[640px]'>
        <DialogHeader>
          <DialogTitle>{t('relaySites.dialogs.importChannel.title')}</DialogTitle>
          <DialogDescription>{t('relaySites.dialogs.importChannel.description')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit, () => {})} noValidate>
          <div className='grid max-h-[72vh] gap-4 overflow-y-auto py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='relay-site-import-channel-name'>{t('relaySites.importChannel.fields.name')}</Label>
              <Input id='relay-site-import-channel-name' {...register('name', { required: t('relaySites.validation.channelNameRequired') })} />
              {errors.name && <span className='text-sm text-red-500'>{errors.name.message}</span>}
            </div>

            <div className='grid gap-2'>
              <Label htmlFor='relay-site-import-channel-type'>{t('relaySites.importChannel.fields.type')}</Label>
              <Select value={watch('type')} onValueChange={(value) => setValue('type', value as ChannelType)}>
                <SelectTrigger id='relay-site-import-channel-type'><SelectValue /></SelectTrigger>
                <SelectContent>
                  {channelTypes.map((config) => (
                    <SelectItem key={config.channelType} value={config.channelType}>
                      {t(`channels.types.${config.channelType}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className='grid gap-2'>
              <Label htmlFor='relay-site-import-channel-base-url'>{t('relaySites.importChannel.fields.baseURL')}</Label>
              <Input
                id='relay-site-import-channel-base-url'
                placeholder='https://relay.example.com/v1'
                {...register('baseURL', { required: t('relaySites.validation.channelBaseURLRequired') })}
              />
              {errors.baseURL && <span className='text-sm text-red-500'>{errors.baseURL.message}</span>}
            </div>

            <div className='grid gap-2'>
              <Label htmlFor='relay-site-import-channel-models'>{t('relaySites.importChannel.fields.supportedModels')}</Label>
              <Textarea
                id='relay-site-import-channel-models'
                rows={5}
                {...register('supportedModels', {
                  validate: (value) => parseList(value).length > 0 || t('relaySites.validation.supportedModelsRequired'),
                })}
              />
              {errors.supportedModels && <span className='text-sm text-red-500'>{errors.supportedModels.message}</span>}
            </div>

            <div className='grid gap-2'>
              <Label htmlFor='relay-site-import-channel-default-model'>{t('relaySites.importChannel.fields.defaultTestModel')}</Label>
              <Input
                id='relay-site-import-channel-default-model'
                {...register('defaultTestModel', { required: t('relaySites.validation.defaultTestModelRequired') })}
              />
              {errors.defaultTestModel && <span className='text-sm text-red-500'>{errors.defaultTestModel.message}</span>}
            </div>

            <div className='grid gap-2'>
              <Label htmlFor='relay-site-import-channel-tags'>{t('relaySites.importChannel.fields.tags')}</Label>
              <Input id='relay-site-import-channel-tags' {...register('tags')} />
            </div>

            <div className='grid gap-2'>
              <Label htmlFor='relay-site-import-channel-remark'>{t('relaySites.importChannel.fields.remark')}</Label>
              <Textarea id='relay-site-import-channel-remark' rows={3} {...register('remark')} />
            </div>
          </div>
          <DialogFooter>
            <Button type='button' variant='outline' onClick={() => setOpen(false)}>{t('common.buttons.cancel')}</Button>
            <Button type='submit' disabled={importMutation.isPending}>
              {importMutation.isPending ? t('common.buttons.saving') : t('relaySites.actions.importChannel')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
