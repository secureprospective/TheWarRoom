// A read session is one account/league/provider lifetime. Ending it is an explicit call by the code that
// changes that identity; it installs no listener, timer or transport request, and never reads again itself.
type Member = { invalidate: () => void; reset: () => void };
// shared: the displayed view stays until its owner rereads; only caches and pending publication are dropped.
// private: the view held another identity's data, so it is cleared back to its initial state.
export type SessionEnd = 'shared' | 'private';

export function createReadSession() {
  const members = new Map<Member, SessionEnd>();
  return {
    join(query: Member, end: SessionEnd) {
      members.set(query, end);
      return () => { members.delete(query); };
    },
    // Members that join while the session is ending belong to the next session.
    end() {
      for (const [query, end] of [...members]) {
        if (end === 'private') query.reset();
        else query.invalidate();
      }
    },
  };
}
export type ReadSession = ReturnType<typeof createReadSession>;
