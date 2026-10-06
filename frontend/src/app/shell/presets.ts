import type { Density } from '../cards/Card';
import type { Route } from './nodes';

export type Preset = {
  label: string;
  route: Route;
  density: Density;
  inspector: 'closed' | 'rest' | 'expanded';
  comms: boolean;
  roles: readonly ('gm' | 'commish' | 'admin')[];
};

export const presets = {
  default: {
    label: 'Default',
    route: { node: 'home', workspace: 'seasonal-card' },
    density: 'tactical',
    inspector: 'rest',
    comms: true,
    roles: ['gm', 'commish', 'admin'],
  },
  draft: {
    label: 'Draft Mode',
    route: { node: 'trade', workspace: 'draft-room' },
    density: 'matrix',
    inspector: 'closed',
    comms: true,
    roles: ['gm', 'commish', 'admin'],
  },
  gameday: {
    label: 'Gameday',
    route: { node: 'pulse', workspace: 'now' },
    density: 'tactical',
    inspector: 'expanded',
    comms: false,
    roles: ['gm', 'commish', 'admin'],
  },
  trade: {
    label: 'Trade Season',
    route: { node: 'trade', workspace: 'trade-desk-and-offers' },
    density: 'tactical',
    inspector: 'expanded',
    comms: true,
    roles: ['gm', 'commish', 'admin'],
  },
  admin: {
    label: 'Admin',
    route: { node: 'control', workspace: 'league' },
    density: 'matrix',
    inspector: 'closed',
    comms: false,
    roles: ['commish', 'admin'],
  },
} as const satisfies Record<string, Preset>;

export type PresetId = keyof typeof presets;
export const presetIds = Object.keys(presets) as PresetId[];

export function presetAllowed(id: PresetId): boolean {
  const preset: Preset = presets[id];
  return preset.roles.includes('gm');
}
