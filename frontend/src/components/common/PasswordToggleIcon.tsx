export function PasswordToggleIcon({ hidden }: { hidden: boolean }) {
  return hidden ? (
    <svg aria-hidden="true" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="m3 3 18 18" />
      <path d="M10.6 10.7a2 2 0 0 0 2.7 2.7" />
      <path d="M9.9 4.2A10.7 10.7 0 0 1 12 4c5.4 0 9 5.5 9 8a9.2 9.2 0 0 1-2.1 3.5" />
      <path d="M6.6 6.6C4.4 8.1 3 10.5 3 12c0 2.5 3.6 8 9 8 1.3 0 2.5-.3 3.6-.8" />
    </svg>
  ) : (
    <svg aria-hidden="true" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M2.5 12s3.5-7 9.5-7 9.5 7 9.5 7-3.5 7-9.5 7-9.5-7Z" />
      <circle cx="12" cy="12" r="2.5" />
    </svg>
  );
}
