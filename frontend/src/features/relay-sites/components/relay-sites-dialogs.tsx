'use client';

import { DeleteRelaySiteDialog } from './delete-relay-site-dialog';
import { RelaySiteAnnouncementsDialog } from './relay-site-announcements-dialog';
import { RelaySiteCheckinLogsDialog } from './relay-site-checkin-logs-dialog';
import { RelaySiteFormDialog } from './relay-site-form-dialog';
import { RelaySiteModelsAndTokensDialog } from './relay-site-models-and-tokens-dialog';

export function RelaySitesDialogs() {
  return (
    <>
      <RelaySiteFormDialog mode='create' />
      <RelaySiteFormDialog mode='edit' />
      <RelaySiteModelsAndTokensDialog />
      <RelaySiteCheckinLogsDialog />
      <RelaySiteAnnouncementsDialog />
      <DeleteRelaySiteDialog />
    </>
  );
}
