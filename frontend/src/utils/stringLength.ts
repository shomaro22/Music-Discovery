export function countUnicodeCharacters(value: string): number {
  return Array.from(value).length;
}

export function countUtf8Bytes(value: string): number {
  return new TextEncoder().encode(value).length;
}
