import type { AlertReading, PulseReading } from './contract';
import { parsePulse } from './parsePulse';
import { parseAlerts } from './parseAlerts';
import { TargetAlerts, TargetPulseNow } from '../../../wailsjs/go/main/App';

export async function pulseNow(): Promise<PulseReading> {
  return parsePulse(await TargetPulseNow());
}
export async function alerts(franchiseId: string): Promise<AlertReading> {
  return parseAlerts(await TargetAlerts(franchiseId));
}
