'use client';

import { Fragment, useMemo, useState } from 'react';
import { format } from 'date-fns';
import { Boxes, ChevronDown, ChevronRight } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { PageInfo } from '@/gql/pagination';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { TableSkeleton } from '@/components/ui/table-skeleton';
import { ServerSidePagination } from '@/components/server-side-pagination';
import { RelaySiteActions } from './relay-site-actions';
import { useRelaySitesContext } from '../context/relay-sites-context';
import { useUpdateRelaySite, useRelaySiteChannels, type RelaySite } from '../data/relay-sites';

interface RelaySitesTableProps {
  data: RelaySite[];
  loading?: boolean;
  pageInfo?: PageInfo;
  pageSize: number;
  totalCount?: number;
  nameFilter: string;
  statusFilter: string;
  canWrite: boolean;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onPageSizeChange: (pageSize: number) => void;
  onNameFilterChange: (filter: string) => void;
  onStatusFilterChange: (status: string) => void;
}

function formatDate(value?: string | null) {
  if (!value) return '-';
  return format(new Date(value), 'yyyy-MM-dd HH:mm:ss');
}

function nodes<T>(connection: { edges?: Array<{ node: T }> | null }) {
  return connection.edges?.map((edge) => edge.node) ?? [];
}

function StatusSwitch({ relaySite, canWrite }: { relaySite: RelaySite; canWrite: boolean }) {
  const updateMutation = useUpdateRelaySite();
  const isArchived = relaySite.status === 'archived';
  const isEnabled = relaySite.status === 'enabled';

  return (
    <Switch
      checked={isEnabled}
      disabled={!canWrite || isArchived || updateMutation.isPending}
      onCheckedChange={(checked) => updateMutation.mutate({
        id: relaySite.id,
        input: { status: checked ? 'enabled' : 'disabled' },
      })}
    />
  );
}

function AutoCheckinSwitch({ relaySite, canWrite }: { relaySite: RelaySite; canWrite: boolean }) {
  const updateMutation = useUpdateRelaySite();
  const isArchived = relaySite.status === 'archived';

  return (
    <Switch
      checked={relaySite.autoCheckinEnabled}
      disabled={!canWrite || isArchived || updateMutation.isPending}
      onCheckedChange={(checked) => updateMutation.mutate({
        id: relaySite.id,
        input: { autoCheckinEnabled: checked },
      })}
    />
  );
}

function BalanceCell({ relaySite }: { relaySite: RelaySite }) {
  const latestBalance = nodes(relaySite.balanceSnapshots)[0];
  if (!latestBalance) return <span className='text-sm text-muted-foreground'>-</span>;
  const balance = Number(latestBalance.balance).toFixed(2);
  return (
    <span className='whitespace-nowrap text-sm font-medium'>
      {balance} {latestBalance.unit}
    </span>
  );
}

function ChannelCountCell({ relaySite, onOpen }: { relaySite: RelaySite; onOpen: () => void }) {
  const { data: channels = [] } = useRelaySiteChannels(relaySite.id);
  return (
    <Button variant='ghost' size='sm' className='h-7 gap-1 px-2' onClick={onOpen}>
      <Boxes className='h-3.5 w-3.5' />
      <span className='text-sm'>{channels.length}</span>
    </Button>
  );
}

function ResourceSummary({ relaySite }: { relaySite: RelaySite }) {
  const { t } = useTranslation();
  const latestBalance = nodes(relaySite.balanceSnapshots)[0];
  const groups = nodes(relaySite.groups);
  const modelPrices = nodes(relaySite.modelPrices);
  const summaryModelPrices = modelPrices.slice(0, 8);
  const hiddenModelCount = Math.max(relaySite.modelPrices.totalCount - summaryModelPrices.length, 0);

  return (
    <div className='grid gap-4 rounded-lg border bg-muted/20 p-4 md:grid-cols-3'>
      <div className='space-y-2'>
        <div className='text-sm font-medium'>{t('relaySites.resources.balance')}</div>
        <div className='text-sm text-muted-foreground'>
          {latestBalance ? `${latestBalance.balance} ${latestBalance.unit}` : t('relaySites.resources.empty')}
        </div>
        <div className='text-xs text-muted-foreground'>{latestBalance ? formatDate(latestBalance.pulledAt) : '-'}</div>
      </div>
      <div className='space-y-2'>
        <div className='text-sm font-medium'>{t('relaySites.resources.groups', { count: relaySite.groups.totalCount })}</div>
        <div className='flex flex-wrap gap-2'>
          {groups.length > 0 ? groups.map((group) => (
            <Badge key={group.id} variant='secondary'>{group.name}{group.ratio != null ? ` · ${group.ratio}` : ''}</Badge>
          )) : <span className='text-sm text-muted-foreground'>{t('relaySites.resources.empty')}</span>}
        </div>
      </div>
      <div className='space-y-2'>
        <div className='text-sm font-medium'>{t('relaySites.resources.models', { count: relaySite.modelPrices.totalCount })}</div>
        <div className='flex flex-wrap gap-2'>
          {summaryModelPrices.length > 0 ? (
            <>
              {summaryModelPrices.map((price) => (
                <Badge key={price.id} variant='outline' className='font-mono'>{price.modelID}</Badge>
              ))}
              {hiddenModelCount > 0 && <Badge variant='secondary'>+{hiddenModelCount}</Badge>}
            </>
          ) : <span className='text-sm text-muted-foreground'>{t('relaySites.resources.empty')}</span>}
        </div>
      </div>
    </div>
  );
}

export function RelaySitesTable({
  data,
  loading,
  pageInfo,
  pageSize,
  totalCount,
  nameFilter,
  statusFilter,
  canWrite,
  onNextPage,
  onPreviousPage,
  onPageSizeChange,
  onNameFilterChange,
  onStatusFilterChange,
}: RelaySitesTableProps) {
  const { t } = useTranslation();
  const { setManagingRelaySite, setIsModelsAndTokensDialogOpen, setViewingAnnouncementsRelaySite, setIsAnnouncementsDialogOpen } = useRelaySitesContext();
  const [expandedIDs, setExpandedIDs] = useState<Set<string>>(new Set());

  const columnsCount = 10;
  const expandedMap = useMemo(() => expandedIDs, [expandedIDs]);
  const toggleExpanded = (id: string) => {
    setExpandedIDs((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleOpenModelsAndTokens = (relaySite: RelaySite) => {
    setManagingRelaySite(relaySite);
    setIsModelsAndTokensDialogOpen(true);
  };

  const handleOpenAnnouncements = (relaySite: RelaySite) => {
    setViewingAnnouncementsRelaySite(relaySite);
    setIsAnnouncementsDialogOpen(true);
  };

  return (
    <div className='flex flex-1 flex-col overflow-hidden'>
      <div className='flex flex-col gap-2 sm:flex-row sm:items-center'>
        <Input placeholder={t('relaySites.filters.searchByName')} value={nameFilter} onChange={(event) => onNameFilterChange(event.target.value)} className='sm:max-w-sm' />
        <Select value={statusFilter} onValueChange={onStatusFilterChange}>
          <SelectTrigger className='sm:w-[180px]'><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value='active'>{t('relaySites.filters.active')}</SelectItem>
            <SelectItem value='enabled'>{t('relaySites.status.enabled')}</SelectItem>
            <SelectItem value='disabled'>{t('relaySites.status.disabled')}</SelectItem>
            <SelectItem value='archived'>{t('relaySites.status.archived')}</SelectItem>
            <SelectItem value='all'>{t('relaySites.filters.all')}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className='shadow-soft relative mt-4 flex-1 overflow-auto rounded-lg border border-[var(--table-border)]'>
        <Table className='border-separate border-spacing-0 bg-[var(--table-background)]'>
          <TableHeader className='sticky top-0 z-20 bg-[var(--table-header)] shadow-sm'>
            <TableRow className='border-0'>
              <TableHead className='w-10 border-0' />
              <TableHead className='border-0'>{t('relaySites.columns.site')}</TableHead>
              <TableHead className='border-0'>{t('relaySites.fields.type')}</TableHead>
              <TableHead className='border-0'>{t('common.columns.status')}</TableHead>
              <TableHead className='border-0'>{t('relaySites.columns.autoCheckin')}</TableHead>
              <TableHead className='border-0'>{t('relaySites.columns.lastCheckinAt')}</TableHead>
              <TableHead className='border-0'>{t('relaySites.columns.lastResult')}</TableHead>
              <TableHead className='border-0'>{t('relaySites.columns.balance')}</TableHead>
              <TableHead className='border-0'>{t('relaySites.columns.channelCount')}</TableHead>
              <TableHead className='w-12 border-0'>{t('common.columns.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableSkeleton rows={pageSize} columns={columnsCount} />
            ) : data.length > 0 ? data.map((relaySite) => {
              const expanded = expandedMap.has(relaySite.id);
              return (
                <Fragment key={relaySite.id}>
                  <TableRow className='border-0 !bg-[var(--table-background)]'>
                    <TableCell className='border-0'>
                      <Button variant='ghost' size='icon' className='h-8 w-8' onClick={() => toggleExpanded(relaySite.id)}>
                        <span className='sr-only'>{t('relaySites.actions.toggleResources')}</span>
                        {expanded ? <ChevronDown className='h-4 w-4' /> : <ChevronRight className='h-4 w-4' />}
                      </Button>
                    </TableCell>
                    <TableCell className='min-w-[260px] border-0'>
                      <div className='flex flex-wrap items-center gap-2 font-medium'>
                        <span>{relaySite.name}</span>
                        {relaySite.hasUnreadAnnouncements && (
                          <Badge
                            variant='destructive'
                            className='cursor-pointer hover:bg-destructive/80'
                            onClick={() => handleOpenAnnouncements(relaySite)}
                          >
                            {t('relaySites.announcements.unread')}
                          </Badge>
                        )}
                      </div>
                      <a
                        href={relaySite.baseURL}
                        target='_blank'
                        rel='noopener noreferrer'
                        className='block truncate font-mono text-xs text-muted-foreground hover:text-foreground hover:underline'
                      >
                        {relaySite.baseURL}
                      </a>
                    </TableCell>
                    <TableCell className='border-0'><Badge variant='outline'>{t(`relaySites.types.${relaySite.type}`)}</Badge></TableCell>
                    <TableCell className='border-0'><StatusSwitch relaySite={relaySite} canWrite={canWrite} /></TableCell>
                    <TableCell className='border-0'><AutoCheckinSwitch relaySite={relaySite} canWrite={canWrite} /></TableCell>
                    <TableCell className='border-0 text-sm text-muted-foreground'>{formatDate(relaySite.lastCheckinAt)}</TableCell>
                    <TableCell className='border-0'>
                      <div className='max-w-[280px] overflow-hidden text-ellipsis whitespace-nowrap text-sm text-muted-foreground'>
                        {relaySite.lastSyncError || relaySite.lastCheckinResult || '-'}
                      </div>
                    </TableCell>
                    <TableCell className='border-0'><BalanceCell relaySite={relaySite} /></TableCell>
                    <TableCell className='border-0'><ChannelCountCell relaySite={relaySite} onOpen={() => handleOpenModelsAndTokens(relaySite)} /></TableCell>
                    <TableCell className='border-0'><RelaySiteActions relaySite={relaySite} canWrite={canWrite} /></TableCell>
                  </TableRow>
                  {expanded && (
                    <TableRow key={`${relaySite.id}-resources`} className='border-0 !bg-[var(--table-background)]'>
                      <TableCell colSpan={columnsCount} className='border-0 px-4 pb-4 pt-0'>
                        <ResourceSummary relaySite={relaySite} />
                      </TableCell>
                    </TableRow>
                  )}
                </Fragment>
              );
            }) : (
              <TableRow className='!bg-[var(--table-background)]'>
                <TableCell colSpan={columnsCount} className='h-24 text-center'>{t('common.noData')}</TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <div className='mt-4 flex-shrink-0'>
        <ServerSidePagination
          pageInfo={pageInfo}
          pageSize={pageSize}
          dataLength={data.length}
          totalCount={totalCount}
          selectedRows={0}
          onNextPage={onNextPage}
          onPreviousPage={onPreviousPage}
          onPageSizeChange={onPageSizeChange}
        />
      </div>
    </div>
  );
}
