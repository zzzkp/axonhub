'use client';

import { useEffect, useMemo, useState } from 'react';
import { format } from 'date-fns';
import { Boxes } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { cn } from '@/lib/utils';
import { useRelaySitesContext } from '../context/relay-sites-context';
import type { RelaySite, RelaySiteModelPrice } from '../data/relay-sites';

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

function defaultGroup(relaySite: RelaySite | null) {
  return nodes(relaySite?.groups)[0]?.name ?? '';
}

export function RelaySiteModelsDialog() {
  const { t } = useTranslation();
  const {
    isModelsDialogOpen,
    setIsModelsDialogOpen,
    viewingModelsRelaySite,
    setViewingModelsRelaySite,
  } = useRelaySitesContext();
  const [selectedGroup, setSelectedGroup] = useState('');

  const groups = useMemo(() => nodes(viewingModelsRelaySite?.groups), [viewingModelsRelaySite]);
  const modelPrices = useMemo(
    () => [...nodes(viewingModelsRelaySite?.modelPrices)].sort((left, right) => left.modelID.localeCompare(right.modelID)),
    [viewingModelsRelaySite]
  );
  const visibleModels = useMemo(() => {
    if (!selectedGroup) return [];
    return modelPrices.filter((model) => modelEnabledForGroup(model, selectedGroup));
  }, [modelPrices, selectedGroup]);

  useEffect(() => {
    if (!isModelsDialogOpen) return;
    setSelectedGroup(defaultGroup(viewingModelsRelaySite));
  }, [isModelsDialogOpen, viewingModelsRelaySite]);

  const setOpen = (open: boolean) => {
    setIsModelsDialogOpen(open);
    if (!open) {
      setViewingModelsRelaySite(null);
      setSelectedGroup('');
    }
  };

  return (
    <Dialog open={isModelsDialogOpen} onOpenChange={setOpen}>
      <DialogContent className='flex h-[82vh] flex-col sm:max-w-[1120px]'>
        <DialogHeader>
          <DialogTitle>{t('relaySites.dialogs.models.title')}</DialogTitle>
          <DialogDescription>{viewingModelsRelaySite?.name ?? t('relaySites.dialogs.models.description')}</DialogDescription>
        </DialogHeader>

        <div className='grid min-h-0 flex-1 gap-4 overflow-hidden lg:grid-cols-[240px_minmax(0,1fr)]'>
          <div className='flex min-h-0 flex-col overflow-hidden rounded-md border'>
            <div className='flex items-center gap-2 border-b px-3 py-2 text-sm font-medium'>
              <Boxes className='h-4 w-4' />
              {t('relaySites.models.groupsTitle')}
            </div>
            <div className='min-h-0 flex-1 space-y-1 overflow-y-auto p-2'>
              {groups.length > 0 ? groups.map((group) => (
                <Button
                  key={group.id}
                  type='button'
                  variant='ghost'
                  className={cn('h-auto w-full justify-between gap-2 px-3 py-2 text-left', selectedGroup === group.name && 'bg-muted')}
                  onClick={() => setSelectedGroup(group.name)}
                >
                  <span className='min-w-0 truncate'>{group.name}</span>
                  {group.ratio != null && <Badge variant='secondary'>{group.ratio}</Badge>}
                </Button>
              )) : (
                <div className='px-3 py-8 text-center text-sm text-muted-foreground'>{t('relaySites.resources.empty')}</div>
              )}
            </div>
          </div>

          <div className='flex min-h-0 flex-col overflow-hidden rounded-md border'>
            <div className='flex min-h-11 flex-wrap items-center justify-between gap-2 border-b px-3 py-2'>
              <div className='min-w-0 text-sm font-medium'>
                {selectedGroup ? t('relaySites.models.groupModels', { group: selectedGroup, count: visibleModels.length }) : t('relaySites.models.noGroupSelected')}
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
                    <TableHead>{t('relaySites.models.columns.endpoints')}</TableHead>
                    <TableHead>{t('relaySites.models.columns.syncedAt')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visibleModels.length > 0 ? visibleModels.map((model) => (
                    <TableRow key={model.id}>
                      <TableCell className='min-w-[220px] font-mono text-sm'>{model.modelID}</TableCell>
                      <TableCell className='whitespace-nowrap text-sm'><Badge variant='secondary'>{billingTypeText(model, t)}</Badge></TableCell>
                      <TableCell className='min-w-[180px] whitespace-nowrap font-mono text-sm'>{priceText(model, t)}</TableCell>
                      <TableCell className='min-w-[140px] text-sm text-muted-foreground'>
                        {model.supportedEndpointTypes.length > 0 ? model.supportedEndpointTypes.join(', ') : '-'}
                      </TableCell>
                      <TableCell className='whitespace-nowrap text-sm text-muted-foreground'>{formatDate(model.syncedAt)}</TableCell>
                    </TableRow>
                  )) : (
                    <TableRow>
                      <TableCell colSpan={5} className='h-24 text-center'>{t('common.noData')}</TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
