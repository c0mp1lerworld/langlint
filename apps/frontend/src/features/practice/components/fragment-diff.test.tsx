import { render, screen } from "@testing-library/react";
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
  target_verb_review: "go → went (pasado simple).",
  lexical_clarification: "Sin cambios léxicos.",
  grammar_explanation: "El pasado simple de 'go' es 'went'.",
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
    expect(screen.getByText("go")).toHaveClass("line-through");
    expect(screen.getByText("went")).toHaveClass("font-semibold");
  });

  it("expande y colapsa la explicación", async () => {
    const user = userEvent.setup();
    render(<ul><FragmentDiff fragment={fragment} /></ul>);

    expect(screen.getByText(/El pasado simple de 'go' es 'went'/)).not.toBeVisible();

    const toggle = screen.getByRole("button", { name: "Ver explicación" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");

    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText(/El pasado simple de 'go' es 'went'/)).toBeVisible();
    expect(screen.getByText("tiempo/concordancia")).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Ocultar explicación" }));
    expect(toggle).toHaveAttribute("aria-expanded", "false");
  });

  it("no tiene violaciones de accesibilidad", async () => {
    const { container } = render(<ul><FragmentDiff fragment={fragment} /></ul>);
    await expectNoA11yViolations(container);
  });
});
