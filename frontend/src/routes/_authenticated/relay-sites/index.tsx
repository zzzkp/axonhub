import { createFileRoute } from '@tanstack/react-router';
import RelaySitesManagement from '@/features/relay-sites';

export const Route = createFileRoute('/_authenticated/relay-sites/')({
  component: RelaySitesManagement,
});
