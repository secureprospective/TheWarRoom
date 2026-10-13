import { Act } from '../commands/Act';
import type { BidFormState, FormField, Preview } from './contract';
import { Rules } from './BidPlans';
import { poolState } from './state';

const edit = (field: FormField) => (value: string | number | boolean) => poolState.edit(field, value);

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="bid-field">
      <span>{label}</span>
      {children}
    </label>
  );
}

// An empty date-and-time box is drawn by the webview as a date that looks filled in, so the
// closing time stays a button until the owner asks to set one.
function ClosingTime({ form, note }: { form: BidFormState; note?: string }) {
  return (
    <div className="bid-field bid-closes" role="group" aria-label="Bidding closes">
      <span>Bidding closes (optional)</span>
      {!form.closesOpen && (
        <Act variant="button"
          verb="market" args={{ field: 'pool-closes-set', value: '' }} label="Set a closing time">
          Set a closing time
        </Act>
      )}
      {form.closesOpen && (
        <div className="bid-closes-set">
          <input
            type="date"
            aria-label="Closing date"
            value={form.closesDate}
            onChange={(e) => edit('closesDate')(e.target.value)}
          />
          <input
            type="time"
            aria-label="Closing time"
            value={form.closesTime}
            onChange={(e) => edit('closesTime')(e.target.value)}
          />
          <Act variant="button"
            verb="market" args={{ field: 'pool-closes-clear', value: '' }} label="Clear the closing time">
            Clear
          </Act>
        </div>
      )}
      {note && <small className="bid-note">{note}</small>}
    </div>
  );
}

function ConditionalDrop({ form, preview }: { form: BidFormState; preview: Preview }) {
  const selected = form.conditionalDrop;
  const over = preview.rules.some((r) => r.rule === 'cap-room' && r.verdict === 'fail');
  if (!selected && !over) return null;
  const chosen = preview.conditionalDrops?.find((p) => p.id === selected);
  return (
    <div className="bid-field">
      {selected ? <span>
        Conditional drop if you win: {chosen?.label ?? selected}{' '}
        <Act variant="text" verb="market" args={{ field: 'pool-conditional-change', value: '' }}>
          Change
        </Act>{' '}
        <Act variant="text" verb="market" args={{ field: 'pool-conditional-clear', value: '' }}>
          Clear
        </Act>
      </span> : <Act variant="text" verb="market"
        args={{ field: 'pool-conditional-change', value: '' }}>
        Over your cap room? Name a conditional drop
      </Act>}
      {form.conditionalOpen && <>
        <select aria-label="Conditional drop if you win" value={selected ?? ''}
          onChange={(e) => edit('conditionalDrop')(e.target.value)}>
          <option value="">Choose one of your rostered players</option>
          {preview.conditionalDrops?.map((p) => (
            <option key={p.id} value={p.id} disabled={!p.available}>{p.label}</option>
          ))}
        </select>
        <small className="bid-note">
          Only if you win; the commissioner makes the drop. Future salaries are held flat;
          restructure history is not held. Nothing is dropped for you.
        </small>
      </>}
    </div>
  );
}

export function PreviewView({ preview, form }: { preview?: Preview; form?: BidFormState }) {
  if (!preview) return <p role="status">Checking the bid rules…</p>;
  if (preview.problem) return <p role="status" className="bid-problem">{preview.problem}</p>;
  return (
    <div className="bid-preview" data-endpoint="C-06.7">
      <dl className="kv">
        <dt>Annual salary</dt>
        <dd className="pool-money">{preview.salary.text}</dd>
        <dt>Bid points</dt>
        <dd>{preview.points}</dd>
      </dl>
      {preview.snapped && <p>{preview.snapped}.</p>}
      <Rules rules={preview.rules} summary={preview.ruleSummary} />
      {form && <ConditionalDrop form={form} preview={preview} />}
      <p role="status">{preview.blocked ? `Blocked: ${preview.note}` : 'Every rule passes.'}</p>
    </div>
  );
}

export function BidForm({ form, preview, busy }: { form: BidFormState; preview?: Preview; busy: string }) {
  const save = form.planId ? 'Save changes' : 'Save bid plan';
  return (
    <section className="bid-form" aria-label="Bid plan form" data-endpoint="M-194">
      <h5>
        {form.planId ? 'Change bid plan' : 'Plan a bid'} · {form.playerName}
      </h5>
      <div className="bid-fields">
        <Field label="Annual salary ($ millions)">
          <input
            inputMode="decimal"
            placeholder="5.25"
            value={form.salary}
            onChange={(e) => edit('salary')(e.target.value)}
          />
        </Field>
        <Field label="Years">
          <select value={form.years} onChange={(e) => edit('years')(Number(e.target.value))}>
            {[1, 2, 3, 4].map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Bid to beat (points, optional)">
          <input
            inputMode="decimal"
            placeholder="6.500"
            value={form.beat}
            onChange={(e) => edit('beat')(e.target.value)}
          />
        </Field>
        <ClosingTime form={form} note={preview?.closesNote} />
        <Field label="Evidence (link or note)">
          <input value={form.evidence} onChange={(e) => edit('evidence')(e.target.value)} />
        </Field>
        <label className="bid-check">
          <input
            type="checkbox"
            checked={form.snipe}
            onChange={(e) => edit('snipe')(e.target.checked)}
          />
          <span>Snipe: this bid lands after the 20-hour mark</span>
        </label>
      </div>
      <p className="bid-note">
        You tell the app about a snipe and the bid to beat; it cannot see the auction clock or other
        teams' bids.
      </p>
      <PreviewView preview={preview} form={form} />
      {preview && !preview.problem && preview.post && (
        <pre className="bid-post" aria-label="ProBoards post preview">
          {preview.post}
        </pre>
      )}
      <div className="bid-actions">
        <Act variant="button"
          verb="market"
          args={{ field: 'pool-save', value: form.planId }}
          label={save}
          disabled={!preview || Boolean(preview.problem) || busy === 'save'}
        >
          {save}
        </Act>
        <Act variant="button"
          verb="market" args={{ field: 'pool-cancel', value: '' }} label="Close the bid form">
          Cancel
        </Act>
      </div>
    </section>
  );
}

