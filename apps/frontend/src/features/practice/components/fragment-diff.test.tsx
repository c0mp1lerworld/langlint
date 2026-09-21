import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import type { components } from "@/lib/api/gen";
import { expectNoA11yViolations } from "@/test/a11y";
import { FragmentDiff } from "./fragment-diff";

type Fragment = components["schemas"]["Fragment"];

const fragment: Fragment = {
  source_es: "Ayer fui al parque.",
  user_draft: "Yesterday I go to the park.",
  correction: "Yesterday I went to the park.",
  target_verb_review: {
    verb: "go",
    correct_form: "went",
    rule: "pasado simple irregular",
    why: "la acción ocurrió ayer",
    es_contrast: "en español el pretérito cambia la forma",
    alternatives: ["went", "did go (énfasis)"],
  },
  lexical_clarification: {
    term: "go",
    meaning: "ir",
    why_wrong: "el borrador usa el presente",
    alternatives: [],
  },
  grammar_explanation: {
    rule_name: "pasado simple irregular",
    explanation: "el verbo no añade -ed, sino que cambia de forma",
    construction: "go → went",
    counterexample: "I go → I went",
    exception: "los regulares añaden -ed",
    es_contrast: "el español usa 'fui'",
  },
  error_patterns: [{ code: "tense_agreement", severity: "critical", note: "pasado" }],
};

describe("FragmentDiff", () => {
  it("renderiza las tres columnas", () => {
    render(<ul><FragmentDiff fragment={fragment} /></ul>);
    expect(screen.getByRole("heading", { name: "Español" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Tu borrador" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Corrección IA" })).toBeInTheDocument();
  });

  it("marca el borrador (del) y la corrección (ins) con doble cue", () => {
    render(<ul><FragmentDiff fragment={fragment} /></ul>);
    const draft = screen.getByRole("region", { name: "Tu borrador" });
    const correction = screen.getByRole("region", { name: "Corrección IA" });
    expect(within(draft).getByText("go")).toHaveClass("line-through");
    expect(within(correction).getByText("went")).toHaveClass("font-semibold");
  });

  it("expande y colapsa las tarjetas de explicación estructurada", async () => {
    const user = userEvent.setup();
    render(<ul><FragmentDiff fragment={fragment} /></ul>);

    expect(screen.getByText(/el verbo no añade -ed/)).not.toBeVisible();

    const toggle = screen.getByRole("button", { name: "Ver explicación" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");

    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");

    expect(screen.getByRole("heading", { name: "Verbo objetivo" })).toBeVisible();
    expect(screen.getByRole("heading", { name: "Aclaración léxica" })).toBeVisible();
    expect(screen.getByRole("heading", { name: "Explicación gramatical" })).toBeVisible();
    expect(screen.getByText(/el verbo no añade -ed/)).toBeVisible();
    expect(screen.getByText("did go (énfasis)")).toBeVisible();
    expect(screen.getByText("tiempo/concordancia")).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Ocultar explicación" }));
    expect(toggle).toHaveAttribute("aria-expanded", "false");
  });

  it("no tiene violaciones de accesibilidad", async () => {
    const { container } = render(<ul><FragmentDiff fragment={fragment} /></ul>);
    await expectNoA11yViolations(container);
  });
});
