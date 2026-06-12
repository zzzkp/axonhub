'use client';

import { useMemo, useState } from 'react';
import { CheckCircle2, XCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { useRelaySitesContext } from '../context/relay-sites-context';
import type { RelaySiteBatchOperationResult } from '../data/relay-sites';

type BatchResultItem = {
  relaySiteID: string;
  relaySiteName: string;
  status: 'success' | 'failed';
  errorMessage?: string;
};

function buildResultItems(result: RelaySiteBatchOperationResult | null): BatchResultItem[] {
  if (!result) return [];

  const items: BatchResultItem[] = [];

  // Add failures
  result.failures.forEach(failure => {
    items.push({
      relaySiteID: failure.relaySiteID,
      relaySiteName: failure.relaySiteName,
      status: 'failed',
      errorMessage: failure.errorMessage,
    });
  });

  // Calculate successful sites (totalCount - failedCount)
  const successCount = result.successCount;
  // We don't have individual success site names, so we'll just show the count in the summary
  // For now, we only show failures in detail

  return items;
}

export function RelaySiteBatchResultDialog() {
  const { t } = useTranslation();
  const {
    isBatchResultDialogOpen,
    setIsBatchResultDialogOpen,
    batchOperationResult,
    setBatchOperationResult,
  } = useRelaySitesContext();

  const [nameFilter, setNameFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'success' | 'failed'>('all');

  const items = useMemo(() => buildResultItems(batchOperationResult), [batchOperationResult]);

  const filteredItems = useMemo(() => {
    return items.filter(item => {
      if (nameFilter && !item.relaySiteName.toLowerCase().includes(nameFilter.toLowerCase())) {
        return false;
      }
      if (statusFilter !== 'all' && item.status !== statusFilter) {
        return false;
      }
      return true;
    });
  }, [items, nameFilter, statusFilter]);

  const setOpen = (open: boolean) => {
    setIsBatchResultDialogOpen(open);
    if (!open) {
      setBatchOperationResult(null);
      setNameFilter('');
      setStatusFilter('all');
    }
  };

  if (!batchOperationResult) return null;

  const { totalCount, successCount, failedCount } = batchOperationResult;

  return (
    <Dialog open={isBatchResultDialogOpen} onOpenChange={setOpen}>
      <DialogContent className='flex max-h-[82vh] flex-col sm:max-w-[860px]'>
        <DialogHeader>
          <DialogTitle>{t('relaySites.batchResult.title')}</DialogTitle>
          <DialogDescription>
            {t('relaySites.batchResult.summary', { success: successCount, failed: failedCount, total: totalCount })}
          </DialogDescription>
        </DialogHeader>

        <div className='flex flex-col gap-2 sm:flex-row sm:items-center'>
          <Input
            placeholder={t('relaySites.filters.searchByName')}
            value={nameFilter}
            onChange={e => setNameFilter(e.target.value)}
            className='sm:max-w-sm'
          />
          <Select value={statusFilter} onValueChange={value => setStatusFilter(value as 'all' | 'success' | 'failed')}>
            <SelectTrigger className='sm:w-[180px]'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='all'>{t('relaySites.filters.all')}</SelectItem>
              <SelectItem value='success'>{t('relaySites.batchResult.status.success')}</SelectItem>
              <SelectItem value='failed'>{t('relaySites.batchResult.status.failed')}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className='min-h-0 flex-1 overflow-auto rounded-md border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('relaySites.batchResult.columns.site')}</TableHead>
                <TableHead>{t('common.columns.status')}</TableHead>
                <TableHead>{t('relaySites.batchResult.columns.error')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {failedCount === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className='h-24 text-center text-muted-foreground'>
                    {t('relaySites.batchResult.allSuccess')}
                  </TableCell>
                </TableRow>
              ) : filteredItems.length > 0 ? (
                filteredItems.map(item => (
                  <TableRow key={item.relaySiteID}>
                    <TableCell className='font-medium'>{item.relaySiteName}</TableCell>
                    <TableCell>
                      {item.status === 'success' ? (
                        <Badge variant='default' className='gap-1'>
                          <CheckCircle2 className='h-3 w-3' />
                          {t('relaySites.batchResult.status.success')}
                        </Badge>
                      ) : (
                        <Badge variant='destructive' className='gap-1'>
                          <XCircle className='h-3 w-3' />
                          {t('relaySites.batchResult.status.failed')}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className='max-w-[400px] text-sm text-muted-foreground'>
                      {item.errorMessage || '-'}
                    </TableCell>
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
      </DialogContent>
    </Dialog>
  );
}
