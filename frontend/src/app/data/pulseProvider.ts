import type { AlertReading } from './contract';
import { parseAlerts } from './parseAlerts';
import { readAlerts } from '../control/api';

export async function alerts(franchiseId: string): Promise<AlertReading> {
  return parseAlerts(await readAlerts(franchiseId));
}
