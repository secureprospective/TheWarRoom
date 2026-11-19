import { useState, type FormEvent } from 'react';
import { pointsLabel } from '../data/nativePulse';
import type { HomeBrief } from './homeReads';
import { HQAct } from './HQAct';

type Answer = { label: 'FROM HOME READS' | 'READING UNAVAILABLE' | 'NOT ANSWERED'; text: string };
const myScoreQuestion = /\b(am i|are we)\b.*\b(win|ahead|lead|los|behind|trail|score|scor)\w*\b/i;
const myMatchupQuestion = /\b(my|our)\s+(score|matchup)\b/i;

/** Answers only score questions from Home's existing reading; unsupported questions stay local. */
export function answerPhoneQuestion(question: string, brief: HomeBrief, mine: string): Answer {
  if (myScoreQuestion.test(question) || myMatchupQuestion.test(question)) {
    if (brief.errors.results) return { label: 'READING UNAVAILABLE',
      text: `The score reading is unavailable · ${brief.errors.results}` };
    const game = brief.results?.games.find(g => g.home === mine || g.away === mine);
    if (!game) return { label: 'READING UNAVAILABLE', text: brief.results
      ? 'No matchup is present in the current score reading.'
      : 'No score reading is held on Home yet.' };
    const score = (franchise: string) => {
      const row = brief.results?.scores.find(item => item.franchise === franchise);
      return row?.points ?? row?.provisional;
    };
    const own = score(mine), opponent = score(game.home === mine ? game.away : game.home);
    if (own === undefined || opponent === undefined)
      return { label: 'READING UNAVAILABLE', text: 'The current matchup score is not in the held reading.' };
    const difference = own - opponent;
    const standing = difference === 0 ? 'level with' : difference > 0 ? 'ahead of' : 'behind';
    const other = game.home === mine ? game.awayName : game.homeName;
    return { label: 'FROM HOME READS',
      text: `You are ${standing} ${other}, ${pointsLabel(own)}–${pointsLabel(opponent)}.` };
  }

  if (/\b(score|scores|matchup|matchups)\b/i.test(question)) {
    if (brief.errors.results) return { label: 'READING UNAVAILABLE',
      text: `The score reading is unavailable · ${brief.errors.results}` };
    const results = brief.results;
    if (!results) return { label: 'READING UNAVAILABLE', text: 'No score reading is held on Home yet.' };
    const lines = results.games.flatMap(game => {
      const home = results.scores.find(row => row.franchise === game.home);
      const away = results.scores.find(row => row.franchise === game.away);
      const homePoints = home?.points ?? home?.provisional;
      const awayPoints = away?.points ?? away?.provisional;
      return homePoints === undefined || awayPoints === undefined ? []
        : [`${game.homeName} ${pointsLabel(homePoints)}–${pointsLabel(awayPoints)} ${game.awayName}`];
    });
    return lines.length ? { label: 'FROM HOME READS', text: `Week ${results.week} · ${lines.join(' · ')}` }
      : { label: 'READING UNAVAILABLE', text: 'No matchup scores are present in the held reading.' };
  }

  return { label: 'NOT ANSWERED',
    text: 'I can answer score questions from the readings already on Today. Nothing was sent or saved.' };
}

export function PhoneAsk({ brief, mine }: { brief: HomeBrief; mine: string }) {
  const [open, setOpen] = useState(false);
  const [question, setQuestion] = useState('');
  const [answer, setAnswer] = useState<Answer>();
  const ask = (text: string) => {
    setQuestion(text);
    setAnswer(answerPhoneQuestion(text, brief, mine));
  };
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (question.trim()) ask(question.trim());
  };

  return <div className="phone-ask">
    <HQAct className="phone-ask__open" label="Ask" expanded={open} onAction={() => setOpen(true)}>
      Ask</HQAct>
    {open && <div className="phone-ask__scrim">
      <section className="phone-ask__sheet" role="dialog" aria-modal="true" aria-labelledby="phone-ask-title">
        <header><h2 id="phone-ask-title">Ask</h2>
          <HQAct label="Close Ask" onAction={() => setOpen(false)}>Close</HQAct></header>
        <p>Answers use only readings already on Today. No question is sent or saved.</p>
        <form onSubmit={submit}>
          <label htmlFor="phone-ask-question">Ask about a score</label>
          <input id="phone-ask-question" autoComplete="off" value={question}
            onChange={event => { setQuestion(event.target.value); setAnswer(undefined); }} />
          <HQAct submit label="Ask">Ask</HQAct>
        </form>
        <div className="phone-ask__suggestions" aria-label="Suggested questions">
          {['Am I winning?', 'Scores right now'].map(text => <HQAct key={text} label={text}
            onAction={() => ask(text)}>{text}</HQAct>)}
        </div>
        {answer && <div className="phone-ask__answer" role="status" aria-live="polite">
          <b>{answer.label}</b><p>{answer.text}</p></div>}
      </section>
    </div>}
  </div>;
}
