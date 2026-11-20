import { pointsLabel } from '../data/nativePulse';
import type { HomeBrief } from './homeReads';

type Answer = { label: 'FROM HOME READS' | 'READING UNAVAILABLE' | 'NOT ANSWERED'; text: string };
const myScoreQuestion = /\b(am i|are we)\b.*\b(win|ahead|lead|los|behind|trail|score|scor)\w*\b/i;
const myMatchupQuestion = /\b(my|our)\s+(score|matchup)\b/i;

/** Answers only score questions from an existing Home reading; unsupported questions stay local. */
export function answerPhoneQuestion(question: string, brief: HomeBrief | undefined,
  mine: string): Answer {
  if (myScoreQuestion.test(question) || myMatchupQuestion.test(question)) {
    if (!brief) return { label: 'READING UNAVAILABLE', text: 'No score reading is held on Today yet.' };
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
    if (!brief) return { label: 'READING UNAVAILABLE', text: 'No score reading is held on Today yet.' };
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
