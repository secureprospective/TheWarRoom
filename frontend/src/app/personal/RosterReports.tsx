import { array } from '../data/parseValues';
import { report, type Report } from '../reports/contract';
import { targetRosterReports } from '../contracts/api';

export async function readRosterReports(mine: string): Promise<Report[]> {
  return array(await targetRosterReports(mine), 'Roster reports', report);
}
