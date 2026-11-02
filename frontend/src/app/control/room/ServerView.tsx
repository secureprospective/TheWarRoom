import { Desk, NotYet, useDefaultInspector, useRoom } from './room';

/** Server: the machine the league runs on. The app holds a backup command and nothing else about the
 *  server yet; every other value says so. */
export function ServerView() {
  const { data } = useRoom();
  const st = data.status;
  useDefaultInspector(
    <>
      <h3>Server</h3>
      <div className="sub">
        The machine the league runs on. Server admin is a separate permission from commissioner (ruling R10).
      </div>
    </>,
  );
  const na = (what: string) => <NotYet key={what} what={what} />;
  return (
    <div className="cr-desk" style={{ gridTemplateColumns: 'repeat(3,1fr)' }}>
      <Desk
        title="STATUS"
        sub={st ? st.state : na('server status')}
        kv={[
          ['Server', na('version')],
          ['Tunnel', na('tunnel')],
          ['Signed-in devices', na('devices')],
          ['Database', na('size')],
        ]}
        acts={[['DOWNLOAD STATUS REPORT', 'status']]}
      />
      <Desk
        title="BACKUPS"
        sub="this machine"
        kv={[
          ['Last', na('backup history')],
          ['Kept', na('retention')],
          ['Next', na('schedule')],
        ]}
        acts={[
          ['BACK UP NOW', 'backup.now'],
          ['RESTORE…', 'restore'],
        ]}
      />
      <Desk
        title="UPGRADES"
        sub="signed bundles only"
        kv={[
          ['Running', na('version')],
          ['Available', na('update check')],
        ]}
        acts={[['UPGRADE', 'upgrade']]}
      />
      <Desk
        title="SIGN-IN HELP"
        sub="for owners who are stuck"
        kv={[
          [
            'Authenticator',
            data.signIns
              ? `optional · ${
                  data.signIns.filter((f) => f.principal.startsWith('acct.') && f.authenticator).length
                } use
          it`
              : (data.errors.signIns ?? '—'),
          ],
          ['Locked out', na('lockouts')],
        ]}
        acts={[
          ['RESET AN AUTHENTICATOR', 'account.totp.reset'],
          ['SIGN OUT ALL DEVICES OF…', 'auth.revoke'],
          ['RECOVERY CODES', 'auth.recovery'],
        ]}
      />
      <Desk
        title="FEEDS"
        sub="score truth ladder"
        kv={[
          ['Source health', st?.health ?? data.errors.status ?? '—'],
          ['State', st?.state ?? '—'],
          ['Gameday', st?.gameday || '—'],
        ]}
        acts={[
          ['RETRY', 'data.refresh'],
          ['SWITCH LIVE PROVIDER', 'data.provider'],
        ]}
      />
      <Desk
        title="ROLES ON THIS SERVER"
        sub="separate from league roles"
        kv={[
          ['Server admin', na('server roles')],
          ['Support', 'none'],
        ]}
        acts={[
          ['GRANT SUPPORT (NOT BUILT)', 'support'],
          ['GRANT SERVER ADMIN (NOT BUILT)', 'operator'],
        ]}
      />
    </div>
  );
}
