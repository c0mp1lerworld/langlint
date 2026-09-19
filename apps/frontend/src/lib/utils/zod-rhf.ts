import type { FieldErrors, FieldValues, Resolver } from "react-hook-form";
import type { ZodType } from "zod";

function assignError(errors: Record<string, unknown>, path: PropertyKey[], message: string): void {
  if (path.length === 0) {
    errors.root = { type: "validation", message };
    return;
  }

  let cursor = errors;
  for (let i = 0; i < path.length - 1; i += 1) {
    const key = String(path[i]);
    const next = cursor[key];
    if (typeof next !== "object" || next === null) {
      cursor[key] = {};
    }
    cursor = cursor[key] as Record<string, unknown>;
  }

  cursor[String(path[path.length - 1])] =
    path.length === 1
      ? {
          type: "validation",
          message,
          root: { type: "validation", message },
        }
      : { type: "validation", message };
}

export function zodResolver<TValues extends FieldValues>(
  schema: ZodType<TValues>,
): Resolver<TValues> {
  return (values) => {
    const parsed = schema.safeParse(values);
    if (parsed.success) {
      return { values: parsed.data, errors: {} };
    }

    const errors: Record<string, unknown> = {};
    for (const issue of parsed.error.issues) {
      assignError(errors, issue.path, issue.message);
    }

    return { values: {}, errors: errors as FieldErrors<TValues> };
  };
}
