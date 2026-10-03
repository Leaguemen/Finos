import type { ReactNode } from 'react';
import { ArrowUpRightIcon } from '@heroicons/react/24/outline';
import { Brand } from './brand';
import { AuthGate } from './auth-provider';
export function AuthShell({ children }: { children: ReactNode }) {
  return <AuthGate guest><main className="auth-layout">
    <section className="story-panel" aria-label="Welcome to Finos"><Brand />
      <div className="story-content">
        <p className="eyebrow"><span />A LITTLE CLARITY. EVERY DAY.</p>
        <h1>Good work starts<br />with a <span>clear mind.</span></h1>
        <p className="story-description">A simpler space for your working day.<br />Less friction. More room to move forward.</p>
        <div className="orbit-art" aria-hidden="true">
          <div className="orbit orbit-one" /><div className="orbit orbit-two" /><div className="orbit orbit-three" />
          <div className="orbit-core"><span>f</span><i /></div>
          <span className="orbit-point point-one" /><span className="orbit-point point-two" />
          <div className="art-label"><span className="tiny-dot" />Everything, in its place.<ArrowUpRightIcon /></div>
        </div>
      </div><div className="story-footer"><span>BUILT FOR YOUR EVERYDAY</span><span>01 — A fresh start</span></div>
    </section>
    <section className="form-panel"><div className="mobile-brand"><Brand /></div>{children}
      <footer className="auth-footer"><span>© {new Date().getFullYear()} P.T. Doval Sejahtera Investa</span><span>Made for a better working day.</span></footer>
    </section>
  </main></AuthGate>;
}
