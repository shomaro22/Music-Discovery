const CSRF_COOKIE_NAME = import.meta.env.VITE_CSRF_COOKIE_NAME ?? "csrf_token";
export function getCsrfToken(): string | null { const item=document.cookie.split(";").map(v=>v.trim()).find(v=>v.startsWith(`${CSRF_COOKIE_NAME}=`)); return item ? decodeURIComponent(item.slice(CSRF_COOKIE_NAME.length+1)) : null; }
