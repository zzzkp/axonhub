'use client';

import { format } from 'date-fns';
import { useTranslation } from 'react-i18next';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { useRelaySitesContext } from '../context/relay-sites-context';

function formatDate(value?: string | null) {
  if (!value) return '-';
  return format(new Date(value), 'yyyy-MM-dd HH:mm:ss');
}

function nodes<T>(connection?: { edges?: Array<{ node: T }> | null } | null) {
  return connection?.edges?.map((edge) => edge.node) ?? [];
}

export function RelaySiteCheckinLogsDialog() {
  const { t } = useTranslation();
  const { isCheckinLogsDialogOpen, setIsCheckinLogsDialogOpen, viewingCheckinLogsRelaySite, setViewingCheckinLogsRelaySite } = useRelaySitesContext();
  const logs = nodes(viewingCheckinLogsRelaySite?.checkinLogs);

  const setOpen = (open: boolean) => {
    setIsCheckinLogsDialogOpen(open);
    if (!open) setViewingCheckinLogsRelaySite(null);
  };

  return (
    <Dialog open={isCheckinLogsDialogOpen} onOpenChange={setOpen}>
      <DialogContent className='sm:max-w-[760px]'>
        <DialogHeader>
          <DialogTitle>{t('relaySites.dialogs.checkinLogs.title')}</DialogTitle>
          <DialogDescription>{viewingCheckinLogsRelaySite?.name ?? t('relaySites.dialogs.checkinLogs.description')}</DialogDescription>
        </DialogHeader>
        <div className='max-h-[64vh] overflow-auto rounded-md border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('relaySites.checkinLogs.executedAt')}</TableHead>
                <TableHead>{t('common.columns.status')}</TableHead>
                <TableHead>{t('relaySites.checkinLogs.message')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.length > 0 ? logs.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className='whitespace-nowrap text-sm text-muted-foreground'>{formatDate(log.executedAt)}</TableCell>
                  <TableCell>
                    <Badge variant={log.status === 'success' ? 'default' : 'destructive'}>{t(`relaySites.checkinStatus.${log.status}`)}</Badge>
                  </TableCell>
                  <TableCell className='min-w-[280px] text-sm text-muted-foreground'>{log.errorMessage || log.message || '-'}</TableCell>
                </TableRow>
              )) : (
                <TableRow>
                  <TableCell colSpan={3} className='h-24 text-center'>{t('common.noData')}</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </DialogContent>
    </Dialog>
  );
}
