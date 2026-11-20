import { array, integer, object, requiredText, text, utc } from '../data/parseValues';

export const ProblemReportCreate = 'feedback.report.create' as const;
export const ProblemReportsRead = 'commish.feedback.read' as const;

export type ProblemReport = {
  id: string; league: string; principal: string; text: string; route: string; appRelease: number;
  viewportWidth: number; userAgent: string; createdAt: string;
};

export function parseProblemReports(value: unknown, league: string): ProblemReport[] {
  const reading = object(value, 'Problem reports', ['league', 'reports']);
  if (text(reading.league, 'Problem reports league') !== league)
    throw new Error('Problem reports belong to another league.');
  return array(reading.reports, 'Problem reports', (value, path) => {
    const row = object(value, path, ['id', 'league', 'principal', 'text', 'route', 'appRelease',
      'viewportWidth', 'userAgent', 'createdAt']);
    if (text(row.league, `${path}.league`) !== league)
      throw new Error(`${path}.league: report belongs to another league`);
    const id = requiredText(row.id, `${path}.id`);
    const principal = requiredText(row.principal, `${path}.principal`);
    const reportText = requiredText(row.text, `${path}.text`);
    const route = requiredText(row.route, `${path}.route`);
    const appRelease = integer(row.appRelease, `${path}.appRelease`, 1);
    const viewportWidth = integer(row.viewportWidth, `${path}.viewportWidth`, 1);
    const userAgent = text(row.userAgent, `${path}.userAgent`);
    if (!/^feedback\.[a-f0-9]{32}$/.test(id) || Array.from(reportText).length > 2000
      || !/^[a-z0-9-]+\/[a-z0-9-]+$/.test(route) || appRelease > 999999 || viewportWidth > 20000
      || new TextEncoder().encode(userAgent).length > 512 || /[\p{Cc}]/u.test(userAgent))
      throw new Error(`${path}: invalid problem report`);
    return {
      id, league, principal, text: reportText, route, appRelease, viewportWidth, userAgent,
      createdAt: utc(row.createdAt, `${path}.createdAt`),
    };
  });
}
