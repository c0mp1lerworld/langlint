import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAnalyzePractice, usePractice } from "@/lib/query/practices";
import { expectNoA11yViolations } from "@/test/a11y";
import { analysis, practice } from "@/test/fixtures";
import { PracticeDiff } from "./practice-diff";

vi.mock("@/lib/query/practices", () => ({
  usePractice: vi.fn(),
  useAnalyzePractice: vi.fn(),
}));

vi.mock("@/lib/query/quiz", () => ({
  useGenerateQuizQuestion: vi.fn(() => ({
    mutate: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
  })),
  useEvaluateQuizAnswer: vi.fn(() => ({
    mutate: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
  })),
}));

function mockPractice(result: unknown): void {
  vi.mocked(usePractice).mockReturnValue(result as ReturnType<typeof usePractice>);
}

function mockAnalyze(result: unknown): void {
  vi.mocked(useAnalyzePractice).mockReturnValue(result as ReturnType<typeof useAnalyzePractice>);
}

const idleAnalyze = { mutate: vi.fn(), isPending: false, isError: false, error: null };

describe("PracticeDiff", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAnalyze(idleAnalyze);
  });

  it("estado draft → dispara el análisis al pulsar", async () => {
    const mutate = vi.fn();
    mockAnalyze({ ...idleAnalyze, mutate });
    mockPractice({
      isPending: false,
      isError: false,
      error: null,
      data: { ...practice({ status: "draft" }), analysis: null },
    });

    const user = userEvent.setup();
    render(<PracticeDiff id={practice().id} />);

    await user.click(screen.getByRole("button", { name: "Analizar" }));
    expect(mutate).toHaveBeenCalledOnce();
  });

  it("estado analyzing → mensaje de espera", () => {
    mockPractice({
      isPending: false,
      isError: false,
      error: null,
      data: { ...practice({ status: "analyzing" }), analysis: null },
    });
    render(<PracticeDiff id={practice().id} />);
    expect(screen.getByRole("status")).toHaveTextContent("Analizando tu borrador");
  });

  it("estado completed → renderiza los fragmentos", () => {
    mockPractice({
      isPending: false,
      isError: false,
      error: null,
      data: { ...practice({ status: "completed" }), analysis: analysis() },
    });
    render(<PracticeDiff id={practice().id} />);
    expect(screen.getByRole("heading", { name: "Español" })).toBeInTheDocument();
  });

  it("estado failed → alerta de fallo", () => {
    mockPractice({
      isPending: false,
      isError: false,
      error: null,
      data: { ...practice({ status: "failed" }), analysis: null },
    });
    render(<PracticeDiff id={practice().id} />);
    expect(screen.getByRole("alert")).toHaveTextContent("El análisis no pudo completarse.");
  });

  it("no tiene violaciones de accesibilidad", async () => {
    mockPractice({
      isPending: false,
      isError: false,
      error: null,
      data: { ...practice({ status: "completed" }), analysis: analysis() },
    });
    const { container } = render(<PracticeDiff id={practice().id} />);
    await expectNoA11yViolations(container);
  });
});
