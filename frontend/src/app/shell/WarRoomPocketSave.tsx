import { useState } from 'react';
import { errorCodeFor, errorMessage } from '../data/errorCodes';
import { FailureAlert } from '../data/FailureAlert';
import { HQAct as WarRoomAct } from './HQAct';
import type { WarRoomContext } from './warroomReads';
import type { WarRoomDeal } from './warroomTradeReads';
import { saveWarRoomDraft, type WarRoomSavedDraft } from './warroomDrafts';

export function WarRoomPocketSave({ context, deal, onSaved }: { context: WarRoomContext;
  deal: WarRoomDeal; onSaved: (draft: WarRoomSavedDraft) => void }) {
  const [label, setLabel] = useState('');
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<{
    error?: string; errorCode?: string; saved?: string;
  }>({});
  const hosted = Boolean(context.native && context.native.owner !== false);
  const canSave = hosted && Boolean(deal.partner) && deal.give.length + deal.get.length > 0;
  return <>
    {hosted && <input aria-label="Saved proposal label" maxLength={160} value={label}
      onChange={event => setLabel(event.target.value)} placeholder="Optional label" />}
    <WarRoomAct className="btn sp" disabled={!canSave || pending}
      title={context.native === undefined ? 'Not available in local mode' : undefined}
      onAction={() => {
        if (!canSave || pending) return;
        setPending(true); setMessage({});
        void saveWarRoomDraft(context, deal, label).then(draft => {
          setPending(false); setMessage({ saved: 'Saved in your private pocket.' }); onSaved(draft);
        }, cause => {
          setPending(false);
          setMessage({ error: errorMessage(cause), errorCode: errorCodeFor(cause) });
        });
      }}>{pending ? 'SAVING…' : 'SAVE TO POCKET'}</WarRoomAct>
    {message.error && <FailureAlert
      message={message.error}
      code={message.errorCode}
      league={context.native?.league}
      screen="War Room · Pocket"
      route="war/pocket"
      port={context.native?.port}
    />}
    {message.saved && <span role="status">{message.saved}</span>}
  </>;
}
