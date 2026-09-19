export type DiffTokenType = "equal" | "del" | "ins";

export interface DiffToken {
  type: DiffTokenType;
  value: string;
}

function tokenize(text: string): string[] {
  return text.match(/\S+\s*/g) ?? [];
}

function append(tokens: DiffToken[], type: DiffTokenType, value: string): void {
  const last = tokens[tokens.length - 1];
  if (last !== undefined && last.type === type) {
    last.value += value;
    return;
  }
  tokens.push({ type, value });
}

export function diffWords(before: string, after: string): DiffToken[] {
  const a = tokenize(before);
  const b = tokenize(after);
  const rows = a.length;
  const cols = b.length;

  const lengths: number[][] = Array.from({ length: rows + 1 }, () =>
    new Array<number>(cols + 1).fill(0),
  );

  for (let i = rows - 1; i >= 0; i -= 1) {
    const current = lengths[i];
    const next = lengths[i + 1];
    if (current === undefined || next === undefined) continue;
    for (let j = cols - 1; j >= 0; j -= 1) {
      current[j] =
        a[i] === b[j] ? (next[j + 1] ?? 0) + 1 : Math.max(next[j] ?? 0, current[j + 1] ?? 0);
    }
  }

  const tokens: DiffToken[] = [];
  let i = 0;
  let j = 0;

  while (i < rows && j < cols) {
    if (a[i] === b[j]) {
      append(tokens, "equal", a[i] ?? "");
      i += 1;
      j += 1;
    } else if ((lengths[i + 1]?.[j] ?? 0) >= (lengths[i]?.[j + 1] ?? 0)) {
      append(tokens, "del", a[i] ?? "");
      i += 1;
    } else {
      append(tokens, "ins", b[j] ?? "");
      j += 1;
    }
  }

  while (i < rows) {
    append(tokens, "del", a[i] ?? "");
    i += 1;
  }
  while (j < cols) {
    append(tokens, "ins", b[j] ?? "");
    j += 1;
  }

  return tokens;
}
