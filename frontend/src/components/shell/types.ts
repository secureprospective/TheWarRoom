// Shell vocabulary shared across the B-1 instrument console. The 6 module ids
// are the locked nav set (Session A / Command Ledger A1); density tiers are the
// Session-A/B curiosity-trigger progression (never a "Pro Mode" toggle).

export type Density = 'narrative' | 'tactical' | 'matrix';

export type ModuleId = 'home' | 'assets' | 'pulse' | 'txn' | 'trade' | 'control';

export interface ModuleDef {
  id: ModuleId;
  label: string;
  hint: string;
}

export const MODULES: ModuleDef[] = [
  { id: 'home', label: 'HOME', hint: 'league landing' },
  { id: 'assets', label: 'ASSETS', hint: 'M1 rankings' },
  { id: 'pulse', label: 'PULSE', hint: 'M2 power' },
  { id: 'txn', label: 'TRANSACT', hint: 'roster ops' },
  { id: 'trade', label: 'TRADE', hint: 'multi-leg builder' },
  { id: 'control', label: 'CONTROL', hint: 'commissioner + dev' },
];
