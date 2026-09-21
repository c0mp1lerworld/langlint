import { expect, test, type Request } from "@playwright/test";

const API = "http://localhost:3100";
const PRACTICE_ID = "01a0ba80-67eb-741c-96a8-f7c3e53ee0ee";
const USER_ID = "0192f3a0-0000-7000-8000-000000000000";

const practice = {
  id: PRACTICE_ID,
  user_id: USER_ID,
  source_text: "El perro escapó.",
  draft_text: "The dog escape.",
  target_rules: [{ verb: "escape", tense: "past simple" }],
  status: "draft",
  created_at: "2026-09-19T10:00:00Z",
  updated_at: "2026-09-19T10:00:00Z",
};

const analysis = {
  id: "11111111-2222-7333-8444-555555555555",
  practice_id: PRACTICE_ID,
  model: "gpt-4o-mini",
  model_version: "gpt-4o-mini-2024-07-18",
  status: "completed",
  fragments: [
    {
      source_es: "El perro escapó.",
      user_draft: "The dog escape.",
      correction: "The dog escaped.",
      target_verb_review: {
        verb: "escape",
        correct_form: "escaped",
        rule: "pasado simple regular",
        why: "la acción ocurrió en el pasado",
        es_contrast: "en español el pretérito cambia la forma",
        alternatives: ["escaped"],
      },
      lexical_clarification: {
        term: "escape",
        meaning: "escapar",
        why_wrong: "el borrador usa el presente",
        alternatives: [],
      },
      grammar_explanation: {
        rule_name: "pasado simple regular",
        explanation: "los verbos regulares añaden -d tras vocal",
        construction: "escape + d",
        counterexample: "escape → escaped",
        exception: "los irregulares no siguen este patrón",
        es_contrast: "el español usa 'escapó'",
      },
      error_patterns: [{ code: "tense_agreement", severity: "critical", note: "pasado" }],
    },
  ],
};

// Next.js serves documents and RSC payloads on the same paths as the API; those
// must reach the framework, not the mock.
function isNextInternal(request: Request): boolean {
  if (request.resourceType() === "document") return true;
  if (request.headers()["rsc"] === "1") return true;
  return new URL(request.url()).searchParams.has("_rsc");
}

test("valida el formulario antes de crear la práctica", async ({ page }) => {
  await page.goto("/practices/new");

  let createCalled = false;
  await page.route(`${API}/practices**`, async (route) => {
    createCalled = true;
    await route.fulfill({ json: {} });
  });

  await page.getByRole("button", { name: "Crear práctica" }).click();

  await expect(page.getByText("El texto en español es obligatorio")).toBeVisible();
  expect(createCalled).toBe(false);
});

test("crea una práctica, la analiza y muestra el diff de 3 columnas", async ({ page }) => {
  let analyzeCalled = false;

  await page.route(`${API}/practices**`, async (route) => {
    const request = route.request();
    if (isNextInternal(request)) return route.continue();

    const url = new URL(request.url());
    const method = request.method();

    if (method === "GET" && url.pathname === "/practices") {
      return route.fulfill({ json: { items: [], total: 0 } });
    }
    if (method === "POST" && url.pathname === "/practices") {
      return route.fulfill({ status: 201, json: practice });
    }
    if (method === "POST" && url.pathname === `/practices/${PRACTICE_ID}/analyze`) {
      analyzeCalled = true;
      return route.fulfill({ status: 202, json: { status: "accepted" } });
    }
    if (method === "GET" && url.pathname === `/practices/${PRACTICE_ID}`) {
      const body = analyzeCalled
        ? { ...practice, status: "completed", analysis }
        : { ...practice, status: "draft", analysis: null };
      return route.fulfill({ json: body });
    }
    return route.continue();
  });

  await page.goto("/practices/new");
  await page.getByLabel("Texto en español").fill("El perro escapó.");
  await page.getByLabel("Tu borrador en inglés").fill("The dog escape.");
  await page.getByLabel("Verbo").fill("escape");
  await page.getByRole("button", { name: "Crear práctica" }).click();

  await expect(page).toHaveURL(new RegExp(`/practices/${PRACTICE_ID}$`));
  await page.getByRole("button", { name: "Analizar" }).click();

  await expect(page.getByRole("heading", { name: "Corrección IA" })).toBeVisible();
  await expect(page.getByText("The dog escaped.")).toBeVisible();
  expect(analyzeCalled).toBe(true);
});
