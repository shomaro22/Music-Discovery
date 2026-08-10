let csrfToken: string | null = null;

export function getCSRFToken(): string | null {
  return csrfToken;
}

export function setCSRFToken(token: string | null): void {
  csrfToken = token;
}

export function clearCSRFToken(): void {
  csrfToken = null;
}
