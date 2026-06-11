import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useDebounce } from '@/hooks/use-debounce';
import { usePaginationSearch } from '@/hooks/use-pagination-search';
import { usePermissions } from '@/hooks/usePermissions';
import { Header } from '@/components/layout/header';
import { Main } from '@/components/layout/main';
import { RelaySitesDialogs } from './components/relay-sites-dialogs';
import { RelaySitesPrimaryButtons } from './components/relay-sites-primary-buttons';
import { RelaySitesTable } from './components/relay-sites-table';
import RelaySitesProvider from './context/relay-sites-context';
import { useRelaySites } from './data/relay-sites';

function RelaySitesContent() {
  const { channelPermissions } = usePermissions();
  const { pageSize, setCursors, setPageSize, resetCursor, paginationArgs } = usePaginationSearch({
    defaultPageSize: 20,
    pageSizeStorageKey: 'relay-sites-table-page-size',
  });
  const [nameFilter, setNameFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('active');
  const debouncedNameFilter = useDebounce(nameFilter, 300);

  const whereClause = (() => {
    const where: Record<string, string | string[]> = {};
    if (debouncedNameFilter) where.nameContainsFold = debouncedNameFilter;
    if (statusFilter === 'active') where.statusIn = ['enabled', 'disabled'];
    else if (statusFilter !== 'all') where.status = statusFilter;
    return Object.keys(where).length > 0 ? where : undefined;
  })();

  const { data, isLoading } = useRelaySites({
    ...paginationArgs,
    where: whereClause,
    orderBy: { field: 'CREATED_AT', direction: 'DESC' },
  });

  const sites = data?.edges?.map((edge) => edge.node) || [];

  const handleNextPage = () => {
    if (data?.pageInfo?.hasNextPage && data?.pageInfo?.endCursor) {
      setCursors(data.pageInfo.startCursor ?? undefined, data.pageInfo.endCursor ?? undefined, 'after');
    }
  };

  const handlePreviousPage = () => {
    if (data?.pageInfo?.hasPreviousPage) {
      setCursors(data.pageInfo.startCursor ?? undefined, data.pageInfo.endCursor ?? undefined, 'before');
    }
  };

  const handleNameFilterChange = (filter: string) => {
    setNameFilter(filter);
    resetCursor();
  };

  const handleStatusFilterChange = (status: string) => {
    setStatusFilter(status);
    resetCursor();
  };

  return (
    <div className='flex flex-1 flex-col overflow-hidden'>
      <RelaySitesTable
        data={sites}
        loading={isLoading}
        pageInfo={data?.pageInfo}
        pageSize={pageSize}
        totalCount={data?.totalCount}
        nameFilter={nameFilter}
        statusFilter={statusFilter}
        canWrite={channelPermissions.canWrite}
        onNextPage={handleNextPage}
        onPreviousPage={handlePreviousPage}
        onPageSizeChange={setPageSize}
        onNameFilterChange={handleNameFilterChange}
        onStatusFilterChange={handleStatusFilterChange}
      />
    </div>
  );
}

export default function RelaySitesManagement() {
  const { t } = useTranslation();
  const { channelPermissions } = usePermissions();
  const { pageSize, setCursors, setPageSize, resetCursor, paginationArgs } = usePaginationSearch({
    defaultPageSize: 20,
    pageSizeStorageKey: 'relay-sites-table-page-size',
  });
  const [nameFilter, setNameFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('active');
  const debouncedNameFilter = useDebounce(nameFilter, 300);

  const whereClause = (() => {
    const where: Record<string, string | string[]> = {};
    if (debouncedNameFilter) where.nameContainsFold = debouncedNameFilter;
    if (statusFilter === 'active') where.statusIn = ['enabled', 'disabled'];
    else if (statusFilter !== 'all') where.status = statusFilter;
    return Object.keys(where).length > 0 ? where : undefined;
  })();

  const { data, isLoading } = useRelaySites({
    ...paginationArgs,
    where: whereClause,
    orderBy: { field: 'CREATED_AT', direction: 'DESC' },
  });

  const sites = data?.edges?.map((edge) => edge.node) || [];

  const handleNextPage = () => {
    if (data?.pageInfo?.hasNextPage && data?.pageInfo?.endCursor) {
      setCursors(data.pageInfo.startCursor ?? undefined, data.pageInfo.endCursor ?? undefined, 'after');
    }
  };

  const handlePreviousPage = () => {
    if (data?.pageInfo?.hasPreviousPage) {
      setCursors(data.pageInfo.startCursor ?? undefined, data.pageInfo.endCursor ?? undefined, 'before');
    }
  };

  const handleNameFilterChange = (filter: string) => {
    setNameFilter(filter);
    resetCursor();
  };

  const handleStatusFilterChange = (status: string) => {
    setStatusFilter(status);
    resetCursor();
  };

  return (
    <RelaySitesProvider>
      <Header fixed>
        <div className='flex w-full flex-1 flex-col gap-2 md:flex-row md:items-center md:justify-between md:gap-0'>
          <div className='min-w-0'>
            <h2 className='text-xl font-bold tracking-tight'>{t('relaySites.title')}</h2>
            <p className='text-sm text-muted-foreground'>{t('relaySites.description')}</p>
          </div>
          <RelaySitesPrimaryButtons canWrite={channelPermissions.canWrite} sites={sites} />
        </div>
      </Header>
      <Main fixed>
        <div className='flex flex-1 flex-col overflow-hidden'>
          <RelaySitesTable
            data={sites}
            loading={isLoading}
            pageInfo={data?.pageInfo}
            pageSize={pageSize}
            totalCount={data?.totalCount}
            nameFilter={nameFilter}
            statusFilter={statusFilter}
            canWrite={channelPermissions.canWrite}
            onNextPage={handleNextPage}
            onPreviousPage={handlePreviousPage}
            onPageSizeChange={setPageSize}
            onNameFilterChange={handleNameFilterChange}
            onStatusFilterChange={handleStatusFilterChange}
          />
        </div>
      </Main>
      <RelaySitesDialogs />
    </RelaySitesProvider>
  );
}
