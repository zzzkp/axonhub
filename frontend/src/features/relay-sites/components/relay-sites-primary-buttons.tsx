'use client';

import { CalendarCheck, Download, ExternalLink, Plus, RefreshCw, Upload } from 'lucide-react';
import { useRef, type ChangeEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { useRelaySitesContext } from '../context/relay-sites-context';
import { useCheckinAllRelaySites, useExportRelaySitesBackup, useImportRelaySitesBackup, useSyncAllRelaySites, type RelaySite } from '../data/relay-sites';

export function RelaySitesPrimaryButtons({ canWrite, sites }: { canWrite: boolean; sites: RelaySite[] }) {
  const { t } = useTranslation();
  const { setIsCreateDialogOpen } = useRelaySitesContext();
  const importFileInputRef = useRef<HTMLInputElement>(null);
  const syncAllMutation = useSyncAllRelaySites();
  const checkinAllMutation = useCheckinAllRelaySites();
  const exportBackupMutation = useExportRelaySitesBackup();
  const importBackupMutation = useImportRelaySitesBackup();

  const handleOpenAllExternalPages = () => {
    const urls = sites.map(s => s.externalCheckinPageURL).filter(Boolean);
    urls.forEach(url => window.open(url!, '_blank'));
  };

  const handleOpenFailedPages = () => {
    const failedSites = sites.filter(s => {
      const latestLog = s.checkinLogs.edges?.[0]?.node;
      return latestLog?.status === 'failed';
    });
    const urls = failedSites.map(s => s.checkinPageURL).filter(Boolean);
    urls.forEach(url => window.open(url!, '_blank'));
  };

  const externalPagesCount = sites.filter(s => s.externalCheckinPageURL).length;
  const failedPagesCount = sites.filter(s => {
    const latestLog = s.checkinLogs.edges?.[0]?.node;
    return latestLog?.status === 'failed' && s.checkinPageURL;
  }).length;

  const handleExportBackup = async () => {
    let payload = '';
    try {
      payload = await exportBackupMutation.mutateAsync();
    } catch {
      return;
    }
    const now = new Date();
    const timestamp = [
      now.getFullYear(),
      String(now.getMonth() + 1).padStart(2, '0'),
      String(now.getDate()).padStart(2, '0'),
      '-',
      String(now.getHours()).padStart(2, '0'),
      String(now.getMinutes()).padStart(2, '0'),
      String(now.getSeconds()).padStart(2, '0'),
    ].join('');
    const blob = new Blob([payload], { type: 'application/json;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `axonhub-relay-sites-backup-${timestamp}.json`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  };

  const handleImportBackupClick = () => {
    importFileInputRef.current?.click();
  };

  const handleImportBackupFile = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;

    const confirmed = window.confirm(t('relaySites.backup.importConfirm'));
    if (!confirmed) return;

    let payload = '';
    try {
      payload = (await file.text()).trim();
    } catch {
      toast.error(t('relaySites.messages.importBackupReadFailed'));
      return;
    }
    if (!payload) {
      toast.error(t('relaySites.messages.importBackupEmptyFile'));
      return;
    }
    try {
      await importBackupMutation.mutateAsync({ payload });
    } catch {
      return;
    }
  };

  if (!canWrite) return null;

  return (
    <div className='flex flex-wrap items-center gap-2'>
      <input ref={importFileInputRef} type='file' accept='application/json,.json' className='hidden' onChange={handleImportBackupFile} />
      <Button variant='outline' onClick={handleExportBackup} disabled={exportBackupMutation.isPending}>
        <Download className='mr-2 h-4 w-4' />
        {t('relaySites.buttons.exportBackup')}
      </Button>
      <Button variant='outline' onClick={handleImportBackupClick} disabled={importBackupMutation.isPending}>
        <Upload className='mr-2 h-4 w-4' />
        {t('relaySites.buttons.importBackup')}
      </Button>
      <Button variant='outline' onClick={() => syncAllMutation.mutate()} disabled={syncAllMutation.isPending}>
        <RefreshCw className={`mr-2 h-4 w-4 ${syncAllMutation.isPending ? 'animate-spin' : ''}`} />
        {t('relaySites.buttons.syncAll')}
      </Button>
      <Button variant='outline' onClick={() => checkinAllMutation.mutate()} disabled={checkinAllMutation.isPending}>
        <CalendarCheck className={`mr-2 h-4 w-4 ${checkinAllMutation.isPending ? 'animate-spin' : ''}`} />
        {t('relaySites.buttons.checkinAll')}
      </Button>
      <Button variant='outline' onClick={handleOpenAllExternalPages} disabled={externalPagesCount === 0}>
        <ExternalLink className='mr-2 h-4 w-4' />
        {t('relaySites.buttons.openAllExternalCheckinPages')}
      </Button>
      <Button variant='outline' onClick={handleOpenFailedPages} disabled={failedPagesCount === 0}>
        <ExternalLink className='mr-2 h-4 w-4' />
        {t('relaySites.buttons.openFailedCheckinPages')}
      </Button>
      <Button onClick={() => setIsCreateDialogOpen(true)}>
        <Plus className='mr-2 h-4 w-4' />
        {t('relaySites.buttons.create')}
      </Button>
    </div>
  );
}
