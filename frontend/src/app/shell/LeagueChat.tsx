import { parseRules } from '../commissioner/rules';
import type { HostedPort } from '../commissioner/hostedRoom';
import { HQAct } from './HQAct';

export async function readChatLink(league: string, port: HostedPort): Promise<string> {
  const response = await port({ command: 'native.rules.read', mode: 'read', arguments: { league } });
  const rows = parseRules(response.data).sections.flatMap(s => s.rows);
  const value = rows.find(r => r.field === 'league.chatLink')?.current;
  return typeof value === 'string' ? value : '';
}

// The league's own chat lives outside TheWarRoom (11-25 ruling): open it in a new tab with no opener.
export function LeagueChat({ link }: { link?: string }) {
  if (!link) return null;
  return <HQAct className="league-chat" label="League chat"
    onAction={() => { window.open(link, '_blank', 'noopener,noreferrer'); }}>
    League chat <small>opens outside TheWarRoom</small>
  </HQAct>;
}
