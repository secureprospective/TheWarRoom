import type { AlertReading } from './contract';
import { parseAlerts } from './parseAlerts';
import { TargetAlerts } from '../../../wailsjs/go/main/App';

export async function alerts(franchiseId: string): Promise<AlertReading> {
  return parseAlerts(await TargetAlerts(franchiseId));
}
