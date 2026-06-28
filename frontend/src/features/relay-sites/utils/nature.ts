export type RelaySiteNature = 'public' | 'semi_public' | 'paid';

export function relaySiteNatureTag(nature: RelaySiteNature) {
  switch (nature) {
    case 'public':
      return '公益站';
    case 'semi_public':
      return '半公益';
    default:
      return '收费站';
  }
}
