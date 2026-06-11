'use client';

import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { useRelaySitesContext } from '../context/relay-sites-context';
import { useDeleteRelaySite } from '../data/relay-sites';

export function DeleteRelaySiteDialog() {
  const { t } = useTranslation();
  const { isDeleteDialogOpen, setIsDeleteDialogOpen, deletingRelaySite, setDeletingRelaySite } = useRelaySitesContext();
  const deleteMutation = useDeleteRelaySite();

  const close = () => {
    setIsDeleteDialogOpen(false);
    setDeletingRelaySite(null);
  };

  return (
    <Dialog
      open={isDeleteDialogOpen}
      onOpenChange={(open) => {
        setIsDeleteDialogOpen(open);
        if (!open) setDeletingRelaySite(null);
      }}
    >
      <DialogContent className='sm:max-w-[480px]'>
        <DialogHeader>
          <DialogTitle>{t('relaySites.dialogs.delete.title')}</DialogTitle>
          <DialogDescription>{t('relaySites.dialogs.delete.description', { name: deletingRelaySite?.name ?? '' })}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type='button' variant='outline' onClick={close}>{t('common.buttons.cancel')}</Button>
          <Button
            type='button'
            variant='destructive'
            disabled={deleteMutation.isPending}
            onClick={async () => {
              if (!deletingRelaySite) return;
              await deleteMutation.mutateAsync(deletingRelaySite.id);
              close();
            }}
          >
            {deleteMutation.isPending ? t('common.buttons.processing') : t('common.buttons.delete')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
