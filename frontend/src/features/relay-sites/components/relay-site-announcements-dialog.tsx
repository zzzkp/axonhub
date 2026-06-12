'use client';

import { useMemo } from 'react';
import { format } from 'date-fns';
import { Megaphone, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { useRelaySitesContext } from '../context/relay-sites-context';
import { useMarkRelaySiteAnnouncementsRead, useRefreshRelaySiteAnnouncements, type RelaySite, type RelaySiteAnnouncement, type RelaySiteAnnouncementResult } from '../data/relay-sites';

function nodes<T>(connection?: { edges?: Array<{ node: T }> | null } | null) {
  return connection?.edges?.map((edge) => edge.node) ?? [];
}

function formatDate(value?: string | null) {
  if (!value) return '-';
  return format(new Date(value), 'yyyy-MM-dd HH:mm:ss');
}

function sortAnnouncements(announcements: RelaySiteAnnouncement[]) {
  return [...announcements].sort((left, right) => {
    const leftTime = new Date(left.publishedAt ?? left.fetchedAt).getTime();
    const rightTime = new Date(right.publishedAt ?? right.fetchedAt).getTime();
    return rightTime - leftTime;
  });
}

function announcementStatus(announcement: RelaySiteAnnouncement, t: (key: string) => string) {
  if (announcement.readAt) {
    return <Badge variant='secondary'>{t('relaySites.announcements.read')}</Badge>;
  }

  return <Badge variant='destructive'>{t('relaySites.announcements.unread')}</Badge>;
}

function mergeAnnouncementResult(relaySite: RelaySite, result: RelaySiteAnnouncementResult): RelaySite {
  return {
    ...relaySite,
    name: result.name,
    hasUnreadAnnouncements: result.hasUnreadAnnouncements,
    announcements: result.announcements,
  };
}

export function RelaySiteAnnouncementsDialog() {
  const { t } = useTranslation();
  const {
    isAnnouncementsDialogOpen,
    setIsAnnouncementsDialogOpen,
    viewingAnnouncementsRelaySite,
    setViewingAnnouncementsRelaySite,
  } = useRelaySitesContext();
  const refreshMutation = useRefreshRelaySiteAnnouncements();
  const markReadMutation = useMarkRelaySiteAnnouncementsRead();

  const announcements = useMemo(
    () => sortAnnouncements(nodes(viewingAnnouncementsRelaySite?.announcements)),
    [viewingAnnouncementsRelaySite]
  );
  const hasUnread = announcements.some((announcement) => !announcement.readAt);
  const isBusy = refreshMutation.isPending || markReadMutation.isPending;

  const setOpen = (open: boolean) => {
    setIsAnnouncementsDialogOpen(open);
    if (!open) setViewingAnnouncementsRelaySite(null);
  };

  const refreshAnnouncements = () => {
    if (!viewingAnnouncementsRelaySite) return;
    refreshMutation.mutate(viewingAnnouncementsRelaySite.id, {
      onSuccess: (result) => setViewingAnnouncementsRelaySite(mergeAnnouncementResult(viewingAnnouncementsRelaySite, result)),
    });
  };

  const markRead = () => {
    if (!viewingAnnouncementsRelaySite) return;
    markReadMutation.mutate(viewingAnnouncementsRelaySite.id, {
      onSuccess: (result) => setViewingAnnouncementsRelaySite(mergeAnnouncementResult(viewingAnnouncementsRelaySite, result)),
    });
  };

  return (
    <Dialog open={isAnnouncementsDialogOpen} onOpenChange={setOpen}>
      <DialogContent className='flex max-h-[82vh] flex-col sm:max-w-[860px]'>
        <DialogHeader>
          <DialogTitle>{t('relaySites.dialogs.announcements.title')}</DialogTitle>
          <DialogDescription>{viewingAnnouncementsRelaySite?.name ?? t('relaySites.dialogs.announcements.description')}</DialogDescription>
        </DialogHeader>

        <div className='flex flex-wrap items-center justify-between gap-2'>
          <div className='flex items-center gap-2 text-sm text-muted-foreground'>
            <Megaphone className='h-4 w-4' />
            {t('relaySites.announcements.total', { count: announcements.length })}
          </div>
          <div className='flex items-center gap-2'>
            <Button type='button' variant='outline' size='sm' disabled={isBusy} onClick={refreshAnnouncements}>
              <RefreshCw className='mr-2 h-4 w-4' />
              {t('relaySites.announcements.refresh')}
            </Button>
            <Button type='button' size='sm' disabled={!hasUnread || isBusy} onClick={markRead}>
              {t('relaySites.announcements.markRead')}
            </Button>
          </div>
        </div>

        <div className='min-h-0 flex-1 overflow-auto rounded-md border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('relaySites.announcements.content')}</TableHead>
                <TableHead>{t('relaySites.announcements.type')}</TableHead>
                <TableHead>{t('common.columns.status')}</TableHead>
                <TableHead>{t('relaySites.announcements.publishedAt')}</TableHead>
                <TableHead>{t('relaySites.announcements.fetchedAt')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {announcements.length > 0 ? announcements.map((announcement) => (
                <TableRow key={announcement.id}>
                  <TableCell className='min-w-[260px] whitespace-pre-wrap text-sm'>{announcement.content}</TableCell>
                  <TableCell className='whitespace-nowrap text-sm text-muted-foreground'>{announcement.type || '-'}</TableCell>
                  <TableCell className='whitespace-nowrap'>{announcementStatus(announcement, t)}</TableCell>
                  <TableCell className='whitespace-nowrap text-sm text-muted-foreground'>{formatDate(announcement.publishedAt)}</TableCell>
                  <TableCell className='whitespace-nowrap text-sm text-muted-foreground'>{formatDate(announcement.fetchedAt)}</TableCell>
                </TableRow>
              )) : (
                <TableRow>
                  <TableCell colSpan={5} className='h-24 text-center'>{t('relaySites.announcements.noAnnouncements')}</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </DialogContent>
    </Dialog>
  );
}
