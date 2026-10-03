'use client';
import { useState, type ButtonHTMLAttributes, type InputHTMLAttributes } from 'react';
import { EyeIcon, EyeSlashIcon } from '@heroicons/react/24/outline';
export function Button({ className = '', ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button className={`finos-button ${className}`} {...props} />;
}
export function Field({ label, hint, id, type, ...props }: InputHTMLAttributes<HTMLInputElement> & { label: string; hint?: string; id: string }) {
  const [visible, setVisible] = useState(false);
  const password = type === 'password';
  return <div className="field"><label htmlFor={id}>{label}</label>
    <div className="input-wrap">
      <input {...props} id={id} type={password && visible ? 'text' : type}
        className={password ? 'password-input' : ''} aria-describedby={hint ? `${id}-hint` : undefined} />
      {password && <button className="password-toggle" type="button" onClick={() => setVisible(!visible)}
        aria-label={visible ? 'Hide password' : 'Show password'} aria-pressed={visible}>
        {visible ? <EyeSlashIcon /> : <EyeIcon />}
      </button>}
    </div>{hint && <p className="field-hint" id={`${id}-hint`}>{hint}</p>}
  </div>;
}
