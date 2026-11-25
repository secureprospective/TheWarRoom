import { parseRules } from '../commissioner/rules';
import type { HostedPort } from '../commissioner/hostedRoom';

export async function readChatLink(league: string, port: HostedPort): Promise<string> {
  const response = await port({ command: 'native.rules.read', mode: 'read', arguments: { league } });
  const rules = parseRules(response.data);
  const value = rules.sections.flatMap(s => s.rows).find(r => r.field === 'league.chatLink')?.current;
  return typeof value === 'string' ? value : '';
}

export function LeagueChat({ link }: { link?: string }) {
  if (!link) return null;
  return <a href={link} target="_blank" rel="noopener noreferrer">League chat{' '}
    <small>opens outside TheWarRoom</small></a>;
}
