import { Component, lazy, Suspense, type ErrorInfo, type ReactNode } from 'react';
import { newClientErrorCode } from './errorCodes';

const FailureAlert = lazy(() => import('./FailureAlert')
  .then(module => ({ default: module.FailureAlert })));

type Props = { children: ReactNode };
type State = { code?: string };

const CRASH_MESSAGE =
  'Something went wrong on this screen. Reload the app, or share the code below.';

export class AppErrorBoundary extends Component<Props, State> {
  state: State = {};

  static getDerivedStateFromError(): State {
    return { code: newClientErrorCode() };
  }

  componentDidCatch(_error: Error, _info: ErrorInfo): void {
    // Failure code is shown to the owner; no silent background telemetry is sent.
  }

  render(): ReactNode {
    if (!this.state.code) return this.props.children;
    const code = this.state.code;
    return <main className="twr-app" style={{ padding: 16 }}>
      <Suspense fallback={
        <div>
          <p role="alert">{CRASH_MESSAGE}</p>
          <code data-testid="failure-code">{code}</code>
        </div>
      }>
        <FailureAlert
          message={CRASH_MESSAGE}
          code={code}
          onReload={() => { window.location.reload(); }}
        />
      </Suspense>
    </main>;
  }
}
