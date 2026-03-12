import adamIdleUrl from './characters/adam-idle.png';
import adamRunUrl from './characters/adam-run.png';
import adamPhoneUrl from './characters/adam-phone.png';
import alexIdleUrl from './characters/alex-idle.png';
import alexRunUrl from './characters/alex-run.png';
import alexPhoneUrl from './characters/alex-phone.png';
import ameliaIdleUrl from './characters/amelia-idle.png';
import ameliaRunUrl from './characters/amelia-run.png';
import ameliaPhoneUrl from './characters/amelia-phone.png';
import bobIdleUrl from './characters/bob-idle.png';
import bobRunUrl from './characters/bob-run.png';
import bobPhoneUrl from './characters/bob-phone.png';
import type { ArenaAvatarVariant } from '../types';

export const arenaAssetManifest = {
  avatars: {
    adam: {
      idle: adamIdleUrl,
      run: adamRunUrl,
      phone: adamPhoneUrl,
    },
    alex: {
      idle: alexIdleUrl,
      run: alexRunUrl,
      phone: alexPhoneUrl,
    },
    amelia: {
      idle: ameliaIdleUrl,
      run: ameliaRunUrl,
      phone: ameliaPhoneUrl,
    },
    bob: {
      idle: bobIdleUrl,
      run: bobRunUrl,
      phone: bobPhoneUrl,
    },
  } satisfies Record<ArenaAvatarVariant, Record<'idle' | 'run' | 'phone', string>>,
};

export const avatarSheetMeta = {
  idle: {
    frameWidth: 16,
    frameHeight: 32,
    frameCount: 24,
  },
  run: {
    frameWidth: 16,
    frameHeight: 32,
    frameCount: 24,
  },
  phone: {
    frameWidth: 16,
    frameHeight: 32,
    frameCount: 9,
  },
} as const;
