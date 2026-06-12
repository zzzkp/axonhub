'use client';

import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { PasswordInput } from '@/components/password-input';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { useRelaySitesContext } from '../context/relay-sites-context';
import { useCreateRelaySite, useUpdateRelaySite, type CreateRelaySiteInput, type UpdateRelaySiteInput } from '../data/relay-sites';

type RelaySiteFormValues = {
  name: string;
  type: 'new_api' | 'sub2api';
  baseURL: string;
  status: 'enabled' | 'disabled' | 'archived';
  autoCheckinEnabled: boolean;
  remark: string;
  checkinPageURL: string;
  externalCheckinPageURL: string;
  authType: 'token' | 'password' | 'jwt';
  token: string;
  userId: string;
  username: string;
  password: string;
  refreshToken: string;
  tokenExpiresAt: string;
};

function normalizeBaseURL(value: string) {
  return value.trim().replace(/\/+$/, '');
}

function buildCredential(data: RelaySiteFormValues, credentialRequired: boolean) {
  if (data.authType === 'jwt') {
    const token = data.token.trim();
    const refreshToken = data.refreshToken.trim();
    const tokenExpiresAt = data.tokenExpiresAt.trim();
    if (!token && !refreshToken && !tokenExpiresAt && !credentialRequired) return undefined;
    return {
      authType: 'jwt' as const,
      token,
      refreshToken,
      ...(tokenExpiresAt ? { tokenExpiresAt: new Date(tokenExpiresAt).toISOString() } : {}),
    };
  }

  if (data.authType === 'token') {
    const token = data.token.trim();
    const userId = Number(data.userId.trim());
    if (!token && !data.userId.trim() && !credentialRequired) return undefined;
    return { authType: 'token' as const, token, userId };
  }

  const username = data.username.trim();
  const password = data.password.trim();
  if (!username && !password && !credentialRequired) return undefined;
  return { authType: 'password' as const, username, password };
}

const emptyRelaySiteFormValues: RelaySiteFormValues = {
  name: '',
  type: 'new_api',
  baseURL: '',
  status: 'enabled',
  autoCheckinEnabled: false,
  remark: '',
  checkinPageURL: '',
  externalCheckinPageURL: '',
  authType: 'token',
  token: '',
  userId: '',
  username: '',
  password: '',
  refreshToken: '',
  tokenExpiresAt: '',
};

export function RelaySiteFormDialog({ mode }: { mode: 'create' | 'edit' }) {
  const { t } = useTranslation();
  const {
    isCreateDialogOpen,
    setIsCreateDialogOpen,
    isEditDialogOpen,
    setIsEditDialogOpen,
    editingRelaySite,
    setEditingRelaySite,
  } = useRelaySitesContext();
  const createMutation = useCreateRelaySite();
  const updateMutation = useUpdateRelaySite();
  const isCreate = mode === 'create';
  const open = isCreate ? isCreateDialogOpen : isEditDialogOpen;
  const pending = isCreate ? createMutation.isPending : updateMutation.isPending;

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = useForm<RelaySiteFormValues>({
    defaultValues: {
      ...emptyRelaySiteFormValues,
    },
  });

  const authType = watch('authType');
  const siteType = watch('type');
  const token = watch('token');
  const userId = watch('userId');
  const username = watch('username');
  const password = watch('password');
  const refreshToken = watch('refreshToken');
  const tokenExpiresAt = watch('tokenExpiresAt');

  useEffect(() => {
    if (siteType === 'sub2api') {
      if (authType !== 'jwt') setValue('authType', 'jwt');
      setValue('autoCheckinEnabled', false);
      setValue('checkinPageURL', '');
      setValue('externalCheckinPageURL', '');
    }
    if (siteType === 'new_api' && authType === 'jwt') {
      setValue('authType', 'token');
    }
  }, [authType, setValue, siteType]);

  useEffect(() => {
    if (!open) return;
    if (isCreate) {
      reset(emptyRelaySiteFormValues);
      return;
    }
    if (editingRelaySite) {
      const credential = editingRelaySite.displayCredential;
      reset({
        name: editingRelaySite.name,
        type: editingRelaySite.type,
        baseURL: editingRelaySite.baseURL,
        status: editingRelaySite.status,
        autoCheckinEnabled: editingRelaySite.autoCheckinEnabled,
        remark: editingRelaySite.remark ?? '',
        checkinPageURL: editingRelaySite.checkinPageURL ?? '',
        externalCheckinPageURL: editingRelaySite.externalCheckinPageURL ?? '',
        authType: credential?.authType ?? 'token',
        token: credential?.token ?? '',
        userId: credential?.userId ? String(credential.userId) : '',
        username: credential?.username ?? '',
        password: credential?.password ?? '',
        refreshToken: credential?.refreshToken ?? '',
        tokenExpiresAt: credential?.tokenExpiresAt ? credential.tokenExpiresAt.slice(0, 16) : '',
      });
    }
  }, [editingRelaySite, isCreate, open, reset]);

  const setOpen = (nextOpen: boolean) => {
    if (isCreate) {
      setIsCreateDialogOpen(nextOpen);
      return;
    }
    setIsEditDialogOpen(nextOpen);
    if (!nextOpen) setEditingRelaySite(null);
  };

  const onSubmit = async (data: RelaySiteFormValues) => {
    const credential = buildCredential(data, isCreate);
    if (isCreate && !credential) return;

    if (isCreate) {
      const input: CreateRelaySiteInput = {
        name: data.name.trim(),
        type: data.type,
        baseURL: normalizeBaseURL(data.baseURL),
        status: data.status,
        autoCheckinEnabled: data.type === 'new_api' && data.autoCheckinEnabled,
        remark: data.remark.trim(),
        checkinPageURL: data.type === 'new_api' ? data.checkinPageURL.trim() : '',
        externalCheckinPageURL: data.type === 'new_api' ? data.externalCheckinPageURL.trim() : '',
        credential: credential!,
      };
      await createMutation.mutateAsync(input);
      setOpen(false);
      return;
    }

    if (!editingRelaySite) return;
    const input: UpdateRelaySiteInput = {
      name: data.name.trim(),
      baseURL: normalizeBaseURL(data.baseURL),
      status: data.status,
      autoCheckinEnabled: data.type === 'new_api' && data.autoCheckinEnabled,
      remark: data.remark.trim(),
      checkinPageURL: data.type === 'new_api' ? data.checkinPageURL.trim() : '',
      externalCheckinPageURL: data.type === 'new_api' ? data.externalCheckinPageURL.trim() : '',
      ...(credential ? { credential } : {}),
    };
    await updateMutation.mutateAsync({ id: editingRelaySite.id, input });
    setOpen(false);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className='sm:max-w-[640px]'>
        <DialogHeader>
          <DialogTitle>{t(isCreate ? 'relaySites.dialogs.create.title' : 'relaySites.dialogs.edit.title')}</DialogTitle>
          <DialogDescription>{t(isCreate ? 'relaySites.dialogs.create.description' : 'relaySites.dialogs.edit.description')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit, () => {})} noValidate>
          <div className='grid max-h-[72vh] gap-4 overflow-y-auto py-4'>
            <div className='grid gap-2'>
              <Label htmlFor={`${mode}-relay-site-name`}>{t('relaySites.fields.name')}</Label>
              <Input id={`${mode}-relay-site-name`} {...register('name', { required: t('relaySites.validation.nameRequired') })} />
              {errors.name && <span className='text-sm text-red-500'>{errors.name.message}</span>}
            </div>
            <div className='grid gap-2'>
              <Label htmlFor={`${mode}-relay-site-base-url`}>{t('relaySites.fields.baseURL')}</Label>
              <Input id={`${mode}-relay-site-base-url`} placeholder={siteType === 'sub2api' ? 'https://sub2api.example.com' : 'https://new-api.example.com'} {...register('baseURL', { required: t('relaySites.validation.baseURLRequired') })} />
              {errors.baseURL && <span className='text-sm text-red-500'>{errors.baseURL.message}</span>}
            </div>
            <div className='grid gap-2'>
              <Label htmlFor={`${mode}-relay-site-type`}>{t('relaySites.fields.type')}</Label>
              <Select value={siteType} onValueChange={(value) => setValue('type', value as RelaySiteFormValues['type'])} disabled={!isCreate}>
                <SelectTrigger id={`${mode}-relay-site-type`}><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value='new_api'>{t('relaySites.types.new_api')}</SelectItem>
                  <SelectItem value='sub2api'>{t('relaySites.types.sub2api')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor={`${mode}-relay-site-status`}>{t('relaySites.fields.status')}</Label>
              <Select value={watch('status')} onValueChange={(value) => setValue('status', value as RelaySiteFormValues['status'])}>
                <SelectTrigger id={`${mode}-relay-site-status`}><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value='enabled'>{t('relaySites.status.enabled')}</SelectItem>
                  <SelectItem value='disabled'>{t('relaySites.status.disabled')}</SelectItem>
                  {!isCreate && <SelectItem value='archived'>{t('relaySites.status.archived')}</SelectItem>}
                </SelectContent>
              </Select>
            </div>
            <div className='flex items-center justify-between gap-4 rounded-md border p-3'>
              <Label htmlFor={`${mode}-relay-site-auto-checkin`} className='cursor-pointer'>
                {t('relaySites.fields.autoCheckinEnabled')}
              </Label>
              <Switch
                id={`${mode}-relay-site-auto-checkin`}
                checked={watch('autoCheckinEnabled')}
                disabled={siteType !== 'new_api'}
                onCheckedChange={(checked) => setValue('autoCheckinEnabled', checked)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor={`${mode}-relay-site-remark`}>{t('relaySites.fields.remark')}</Label>
              <Textarea id={`${mode}-relay-site-remark`} rows={3} {...register('remark')} />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor={`${mode}-relay-site-checkin-page-url`}>{t('relaySites.fields.checkinPageURL')}</Label>
              <Input id={`${mode}-relay-site-checkin-page-url`} placeholder='https://...' disabled={siteType !== 'new_api'} {...register('checkinPageURL')} />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor={`${mode}-relay-site-external-checkin-page-url`}>{t('relaySites.fields.externalCheckinPageURL')}</Label>
              <Input id={`${mode}-relay-site-external-checkin-page-url`} placeholder='https://...' disabled={siteType !== 'new_api'} {...register('externalCheckinPageURL')} />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor={`${mode}-relay-site-auth-type`}>{t('relaySites.fields.authType')}</Label>
              <Select value={authType} onValueChange={(value) => setValue('authType', value as RelaySiteFormValues['authType'])}>
                <SelectTrigger id={`${mode}-relay-site-auth-type`}><SelectValue /></SelectTrigger>
                <SelectContent>
                  {siteType === 'sub2api' ? (
                    <SelectItem value='jwt'>{t('relaySites.authTypes.jwt')}</SelectItem>
                  ) : (
                    <>
                      <SelectItem value='token'>{t('relaySites.authTypes.token')}</SelectItem>
                      <SelectItem value='password'>{t('relaySites.authTypes.password')}</SelectItem>
                    </>
                  )}
                </SelectContent>
              </Select>
            </div>
            {authType === 'jwt' ? (
              <div className='grid gap-4'>
                <div className='grid gap-2'>
                  <Label htmlFor={`${mode}-relay-site-token`}>{t('relaySites.fields.jwtToken')}</Label>
                  <PasswordInput id={`${mode}-relay-site-token`} autoComplete='off' className='[&_input]:pr-9' {...register('token', { validate: (value) => (isCreate || refreshToken.trim() || tokenExpiresAt.trim()) ? Boolean(value.trim()) || t('relaySites.validation.jwtTokenRequired') : true })} />
                  {errors.token && <span className='text-sm text-red-500'>{errors.token.message}</span>}
                </div>
                <div className='grid gap-4 sm:grid-cols-2'>
                  <div className='grid gap-2'>
                    <Label htmlFor={`${mode}-relay-site-refresh-token`}>{t('relaySites.fields.refreshToken')}</Label>
                    <PasswordInput id={`${mode}-relay-site-refresh-token`} autoComplete='off' className='[&_input]:pr-9' {...register('refreshToken')} />
                  </div>
                  <div className='grid gap-2'>
                    <Label htmlFor={`${mode}-relay-site-token-expires-at`}>{t('relaySites.fields.tokenExpiresAt')}</Label>
                    <Input id={`${mode}-relay-site-token-expires-at`} type='datetime-local' autoComplete='off' {...register('tokenExpiresAt')} />
                  </div>
                </div>
              </div>
            ) : authType === 'token' ? (
              <div className='grid gap-4 sm:grid-cols-2'>
                <div className='grid gap-2'>
                  <Label htmlFor={`${mode}-relay-site-token`}>{t('relaySites.fields.token')}</Label>
                  <PasswordInput id={`${mode}-relay-site-token`} autoComplete='off' className='[&_input]:pr-9' {...register('token', { validate: (value) => (isCreate || userId.trim()) ? Boolean(value.trim()) || t('relaySites.validation.tokenRequired') : true })} />
                  {errors.token && <span className='text-sm text-red-500'>{errors.token.message}</span>}
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor={`${mode}-relay-site-user-id`}>{t('relaySites.fields.userId')}</Label>
                  <Input id={`${mode}-relay-site-user-id`} inputMode='numeric' autoComplete='off' {...register('userId', { validate: (value) => (isCreate || token.trim()) ? Number(value.trim()) > 0 || t('relaySites.validation.userIdRequired') : true })} />
                  {errors.userId && <span className='text-sm text-red-500'>{errors.userId.message}</span>}
                </div>
              </div>
            ) : (
              <div className='grid gap-4 sm:grid-cols-2'>
                <div className='grid gap-2'>
                  <Label htmlFor={`${mode}-relay-site-username`}>{t('relaySites.fields.username')}</Label>
                  <Input id={`${mode}-relay-site-username`} autoComplete='off' {...register('username', { validate: (value) => (isCreate || password.trim()) ? Boolean(value.trim()) || t('relaySites.validation.usernameRequired') : true })} />
                  {errors.username && <span className='text-sm text-red-500'>{errors.username.message}</span>}
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor={`${mode}-relay-site-password`}>{t('relaySites.fields.password')}</Label>
                  <Input id={`${mode}-relay-site-password`} type='password' autoComplete='off' {...register('password', { validate: (value) => (isCreate || username.trim()) ? Boolean(value.trim()) || t('relaySites.validation.passwordRequired') : true })} />
                  {errors.password && <span className='text-sm text-red-500'>{errors.password.message}</span>}
                </div>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button type='button' variant='outline' onClick={() => setOpen(false)}>{t('common.buttons.cancel')}</Button>
            <Button type='submit' disabled={pending}>{pending ? t('common.buttons.saving') : t(isCreate ? 'common.buttons.create' : 'common.buttons.saveChanges')}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
