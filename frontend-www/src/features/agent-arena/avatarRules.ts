import type { ArenaAvatarVariant } from './types';

export const ARENA_AVATAR_VARIANTS: ArenaAvatarVariant[] = ['adam', 'alex', 'amelia', 'bob'];

function hashSeed(input: string) {
  let value = 0;
  for (let index = 0; index < input.length; index += 1) {
    value = (value * 31 + input.charCodeAt(index)) >>> 0;
  }
  return value;
}

export function isArenaAvatarVariant(value: string | null | undefined): value is ArenaAvatarVariant {
  return Boolean(value && ARENA_AVATAR_VARIANTS.includes(value as ArenaAvatarVariant));
}

export function resolveArenaAvatarVariant(agentId: string, preferredVariant?: string | null) {
  if (isArenaAvatarVariant(preferredVariant)) {
    return preferredVariant;
  }

  const index = hashSeed(agentId) % ARENA_AVATAR_VARIANTS.length;
  return ARENA_AVATAR_VARIANTS[index];
}
