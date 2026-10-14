import { ReportView } from '../reports/ReportView';
import { personLabel } from './forms';
import type { People, PeopleVote } from './people';
import { invitesReport, rolesReport, seatName, seatsReport } from './peopleReport';
import { leagueState } from './proxy';
import './commissioner.css';

const unset = (at: string) => at === '' || at.startsWith('0001-');
const word = (s: string) => personLabel(s).replace(/^./, ch => ch.toUpperCase());

function Vote({ vote, people }: { vote: PeopleVote; people: People }) {
  return (
    <li>
      <strong>{vote.status}</strong>
      <p>{vote.ballotsCast} of {vote.totalFranchises} franchises have voted. {vote.requirement}:{' '}
        {vote.votesRequired}.</p>
      {vote.counts.length > 0 &&
        <p>{vote.counts.map(c => `${word(c.choice)}: ${c.ballots}`).join(' · ')}</p>}
      {vote.candidates.length > 0 && <p>Candidates: {vote.candidates.map(personLabel).join(', ')}</p>}
      {!unset(vote.endsAt) && <p>Voting ends <time dateTime={vote.endsAt}>
        {new Date(vote.endsAt).toLocaleString()}</time></p>}
      {vote.votedFranchises.length > 0 &&
        <p>Voted: {vote.votedFranchises.map(f => seatName(people, f)).join(', ')}</p>}
    </li>
  );
}

// LeaguePeople is the read screen for seats, roles, the trade review group (DOT), votes and succession.
export default function LeaguePeople() {
  const { people, pending } = leagueState.use();
  if (!people) return pending ? <p role="status">Reading league people…</p> : null;
  return (
    <section className="league-people" aria-label="League people">
      <h4>People</h4>
      <p>You are: {people.yourRole}</p>
      <ReportView report={seatsReport(people)} howLabel="How to read this" />
      <p>Trade review seats (DOT): {people.dotCount}. {people.dotStatus}</p>
      {people.roles.length > 0 && <ReportView report={rolesReport(people)} howLabel="How to read this" />}
      {people.invites.length > 0 && <ReportView report={invitesReport(people)} howLabel="How to read this" />}
      <p>{people.succession}</p>
      {people.votes.length > 0 && (
        <section aria-label="Open votes">
          <h5>Open votes</h5>
          <ol className="league-votes">
            {people.votes.map(v => <Vote key={v.id} vote={v} people={people} />)}
          </ol>
        </section>
      )}
    </section>
  );
}
