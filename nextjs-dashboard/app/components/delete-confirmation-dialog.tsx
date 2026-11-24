'use client';

import { useEffect, useRef } from 'react';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { Button } from './ui';

type DeleteConfirmationDialogProps = {
  title: string;
  description: string;
  busy: boolean;
  error?: string;
  onCancel: () => void;
  onConfirm: () => void;
};

export function DeleteConfirmationDialog({ title, description, busy, error, onCancel, onConfirm }: DeleteConfirmationDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (dialog && !dialog.open) dialog.showModal();
    return () => { if (dialog?.open) dialog.close(); };
  }, []);

  return <dialog ref={dialogRef} className="delete-dialog" aria-labelledby="delete-dialog-title" aria-describedby="delete-dialog-description"
    onCancel={event => { event.preventDefault(); if (!busy) onCancel(); }}
    onClick={event => { if (event.target === event.currentTarget && !busy) onCancel(); }}>
    <div className="delete-dialog-content">
      <div className="delete-dialog-icon"><ExclamationTriangleIcon aria-hidden="true" /></div>
      <h2 id="delete-dialog-title">{title}</h2>
      <p id="delete-dialog-description">{description}</p>
      {error && <p className="form-error" role="alert">{error}</p>}
      <div className="delete-dialog-actions">
        <Button type="button" className="button-secondary" autoFocus disabled={busy} onClick={onCancel}>Cancel</Button>
        <Button type="button" className="delete-dialog-confirm" disabled={busy} aria-busy={busy} onClick={onConfirm}>
          {busy ? 'Deleting…' : 'Delete permanently'}
        </Button>
      </div>
    </div>
  </dialog>;
}
