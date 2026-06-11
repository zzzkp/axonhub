'use client';

import { DeleteRelaySiteDialog } from './delete-relay-site-dialog';
import { ImportRelaySiteChannelDialog } from './import-relay-site-channel-dialog';
import { RelaySiteAnnouncementsDialog } from './relay-site-announcements-dialog';
import { RelaySiteAPIKeysDialog } from './relay-site-api-keys-dialog';
import { RelaySiteCheckinLogsDialog } from './relay-site-checkin-logs-dialog';
import { RelaySiteFormDialog } from './relay-site-form-dialog';
import { RelaySiteModelsDialog } from './relay-site-models-dialog';

export function RelaySitesDialogs() {
  return (
    <>
      <RelaySiteFormDialog mode='create' />
      <RelaySiteFormDialog mode='edit' />
      <RelaySiteAPIKeysDialog />
      <RelaySiteModelsDialog />
      <RelaySiteCheckinLogsDialog />
      <RelaySiteAnnouncementsDialog />
      <ImportRelaySiteChannelDialog />
      <DeleteRelaySiteDialog />
    </>
  );
}
