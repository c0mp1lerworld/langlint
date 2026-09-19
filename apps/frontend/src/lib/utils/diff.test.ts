import { describe, expect, it } from "vitest";
import { diffWords, type DiffToken } from "./diff";

function reconstruct(tokens: DiffToken[], side: "before" | "after"): string {
  const keep = side === "before" ? new Set(["equal", "del"]) : new Set(["equal", "ins"]);
  return tokens
    .filter((token) => keep.has(token.type))
    .map((token) => token.value)
    .join("");
}

function assertNoAdjacentSameType(tokens: DiffToken[]): void {
  for (let i = 1; i < tokens.length; i += 1) {
    expect(tokens[i]?.type).not.toBe(tokens[i - 1]?.type);
  }
}

describe("diffWords", () => {
  it("identidad → un único token equal", () => {
    const tokens = diffWords("hello world", "hello world");
    expect(tokens).toEqual([{ type: "equal", value: "hello world" }]);
  });

  it("reemplazo de palabra → del + ins entre los iguales", () => {
    const tokens = diffWords("I go", "I went");
    expect(tokens).toEqual([
      { type: "equal", value: "I " },
      { type: "del", value: "go" },
      { type: "ins", value: "went" },
    ]);
  });

  it("inserción → token ins entre iguales", () => {
    const tokens = diffWords("run fast", "run very fast");
    expect(tokens).toEqual([
      { type: "equal", value: "run " },
      { type: "ins", value: "very " },
      { type: "equal", value: "fast" },
    ]);
  });

  it("eliminación → token del entre iguales", () => {
    const tokens = diffWords("run very fast", "run fast");
    expect(tokens).toEqual([
      { type: "equal", value: "run " },
      { type: "del", value: "very " },
      { type: "equal", value: "fast" },
    ]);
  });

  it("vacíos → solo ins o del", () => {
    expect(diffWords("", "hello")).toEqual([{ type: "ins", value: "hello" }]);
    expect(diffWords("hello", "")).toEqual([{ type: "del", value: "hello" }]);
    expect(diffWords("", "")).toEqual([]);
  });

  it("múltiples cambios → reconstruye ambos lados", () => {
    const before = "Yesterday I run across all the park with my dog.";
    const after = "Yesterday I ran across the whole park with my dog.";
    const tokens = diffWords(before, after);
    expect(reconstruct(tokens, "before")).toBe(before);
    expect(reconstruct(tokens, "after")).toBe(after);
    assertNoAdjacentSameType(tokens);
  });

  it("puntuación y espaciado → invariante de reconstrucción", () => {
    const before = "hello, world!  bye";
    const after = "hello world, bye";
    const tokens = diffWords(before, after);
    expect(reconstruct(tokens, "before")).toBe(before);
    expect(reconstruct(tokens, "after")).toBe(after);
    assertNoAdjacentSameType(tokens);
  });
});
