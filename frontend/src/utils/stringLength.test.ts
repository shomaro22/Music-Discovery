import { describe, expect, it } from "vitest";
import { countUnicodeCharacters, countUtf8Bytes } from "./stringLength";

describe("string length utilities", () => {
  it("counts Unicode characters instead of UTF-16 code units", () => {
    expect(countUnicodeCharacters("🎵")).toBe(1);
    expect("🎵".length).toBe(2);
  });

  it("counts UTF-8 bytes", () => {
    expect(countUtf8Bytes("a")).toBe(1);
    expect(countUtf8Bytes("あ")).toBe(3);
    expect(countUtf8Bytes("🎵")).toBe(4);
  });
});
