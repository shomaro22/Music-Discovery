export function getCSRFCookie(): string | null {
  const cookie = document.cookie
    .split(";")
    .map((value) => value.trim())
    .find((value) => value.startsWith("csrf_token="));

  return cookie ? decodeURIComponent(cookie.slice("csrf_token=".length)) : null;
}
