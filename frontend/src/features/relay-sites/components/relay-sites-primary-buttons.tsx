'use client';

import { CalendarCheck, ExternalLink, Plus, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { useRelaySitesContext } from '../context/relay-sites-context';
import { useCheckinAllRelaySites, useSyncAllRelaySites, type RelaySite } from '../data/relay-sites';

export function RelaySitesPrimaryButtons({ canWrite, sites }: { canWrite: boolean; sites: RelaySite[] }) {
  const { t } = useTranslation();
  const { setIsCreateDialogOpen } = useRelaySitesContext();
  const syncAllMutation = useSyncAllRelaySites();
  const checkinAllMutation = useCheckinAllRelaySites();

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

  if (!canWrite) return null;

  return (
    <div className='flex flex-wrap items-center gap-2'>
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
