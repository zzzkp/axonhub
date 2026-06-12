'use client';

import { Boxes, CalendarCheck, History, KeyRound, Megaphone, MoreHorizontal, Pencil, RefreshCw, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { useRelaySitesContext } from '../context/relay-sites-context';
import { useCheckinRelaySite, useSyncRelaySite, type RelaySite } from '../data/relay-sites';

interface RelaySiteActionsProps {
  relaySite: RelaySite;
  canWrite: boolean;
}

export function RelaySiteActions({ relaySite, canWrite }: RelaySiteActionsProps) {
  const { t } = useTranslation();
  const {
    setEditingRelaySite,
    setIsEditDialogOpen,
    setDeletingRelaySite,
    setIsDeleteDialogOpen,
    setManagingRelaySite,
    setIsModelsAndTokensDialogOpen,
    setViewingCheckinLogsRelaySite,
    setIsCheckinLogsDialogOpen,
    setViewingAnnouncementsRelaySite,
    setIsAnnouncementsDialogOpen,
  } = useRelaySitesContext();
  const syncMutation = useSyncRelaySite();
  const checkinMutation = useCheckinRelaySite();
  const supportsCheckin = relaySite.type === 'new_api';

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant='ghost' className='h-8 w-8 p-0'>
          <span className='sr-only'>{t('common.buttons.openMenu')}</span>
          <MoreHorizontal className='h-4 w-4' />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end'>
        {canWrite && (
          <>
            <DropdownMenuItem disabled={syncMutation.isPending} onClick={() => syncMutation.mutate(relaySite.id)}>
              <RefreshCw className='mr-2 h-4 w-4' />
              {t('relaySites.actions.sync')}
            </DropdownMenuItem>
            {supportsCheckin && (
              <DropdownMenuItem disabled={checkinMutation.isPending} onClick={() => checkinMutation.mutate(relaySite.id)}>
                <CalendarCheck className='mr-2 h-4 w-4' />
                {t('relaySites.actions.checkin')}
              </DropdownMenuItem>
            )}
          </>
        )}
        <DropdownMenuItem
          onClick={() => {
            setManagingRelaySite(relaySite);
            setIsModelsAndTokensDialogOpen(true);
          }}
        >
          <Boxes className='mr-2 h-4 w-4' />
          {t('relaySites.actions.modelsAndTokens')}
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => {
            setViewingCheckinLogsRelaySite(relaySite);
            setIsCheckinLogsDialogOpen(true);
          }}
        >
          <History className='mr-2 h-4 w-4' />
          {t('relaySites.actions.checkinLogs')}
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => {
            setViewingAnnouncementsRelaySite(relaySite);
            setIsAnnouncementsDialogOpen(true);
          }}
        >
          <Megaphone className='mr-2 h-4 w-4' />
          {t('relaySites.actions.announcements')}
        </DropdownMenuItem>
        {canWrite && (
          <>
            <DropdownMenuItem
              onClick={() => {
                setEditingRelaySite(relaySite);
                setIsEditDialogOpen(true);
              }}
            >
              <Pencil className='mr-2 h-4 w-4' />
              {t('common.buttons.edit')}
            </DropdownMenuItem>
            <DropdownMenuItem
              className='text-destructive focus:text-destructive'
              onClick={() => {
                setDeletingRelaySite(relaySite);
                setIsDeleteDialogOpen(true);
              }}
            >
              <Trash2 className='mr-2 h-4 w-4' />
              {t('common.buttons.delete')}
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
