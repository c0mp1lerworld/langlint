import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api/errors";
import { useErrorPatternStats, useProgressSeries } from "@/lib/query/analytics";
import { useAnalyticsStore } from "@/lib/store/analytics";
import { expectNoA11yViolations } from "@/test/a11y";
import { errorPatternAggregate } from "@/test/fixtures";
import { AnalyticsDashboard } from "./analytics-dashboard";

vi.mock("@/lib/query/analytics", () => ({
  useErrorPatternStats: vi.fn(),
  useProgressSeries: vi.fn(),
}));

function mockStats(result: unknown): void {
  vi.mocked(useErrorPatternStats).mockReturnValue(result as ReturnType<typeof useErrorPatternStats>);
}

function mockProgress(result: unknown): void {
  vi.mocked(useProgressSeries).mockReturnValue(result as ReturnType<typeof useProgressSeries>);
}

const statsData = {
  isPending: false,
  isError: false,
  error: null,
  data: { window: "week", patterns: [errorPatternAggregate({ code: "tense_agreement", count: 2 })] },
};

const progressEmpty = {
  isPending: false,
  isError: false,
  error: null,
  data: { window: "week", points: [] },
};

describe("AnalyticsDashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAnalyticsStore.setState({ window: "week" });
    mockStats(statsData);
    mockProgress(progressEmpty);
  });

  it("muestra los estados de carga", () => {
    mockStats({ isPending: true, isError: false, error: null, data: undefined });
    mockProgress({ isPending: true, isError: false, error: null, data: undefined });
    render(<AnalyticsDashboard />);
    expect(screen.getByText("Cargando patrones de error…")).toBeInTheDocument();
    expect(screen.getByText("Cargando progreso…")).toBeInTheDocument();
  });

  it("muestra la alerta del error recurrente y la frecuencia", () => {
    render(<AnalyticsDashboard />);
    expect(screen.getByText(/Tu error más frecuente es/)).toBeInTheDocument();
    expect(screen.getAllByText("tiempo/concordancia").length).toBeGreaterThan(0);
  });

  it("muestra la serie de progreso cuando hay datos", () => {
    mockProgress({
      isPending: false,
      isError: false,
      error: null,
      data: {
        window: "week",
        points: [
          {
            period_start: "2026-09-14T00:00:00Z",
            total_fragments: 10,
            error_count: 2,
            accuracy: 0.8,
          },
        ],
      },
    });
    render(<AnalyticsDashboard />);
    expect(screen.getByLabelText("Serie temporal de progreso")).toBeInTheDocument();
    expect(screen.getByText(/80% precisión/)).toBeInTheDocument();
  });

  it("mapea otros errores de query a UX", () => {
    mockStats({
      isPending: false,
      isError: true,
      error: new ApiError({ status: 503, code: "llm_unavailable", message: "x" }),
      data: undefined,
    });
    render(<AnalyticsDashboard />);
    expect(screen.getByRole("alert")).toHaveTextContent("El motor de análisis no responde");
  });

  it("el selector de ventana actualiza el filtro de la query", async () => {
    const user = userEvent.setup();
    render(<AnalyticsDashboard />);

    await user.click(screen.getByRole("button", { name: "Día" }));

    expect(useAnalyticsStore.getState().window).toBe("day");
    expect(vi.mocked(useErrorPatternStats).mock.calls.at(-1)?.[0]).toBe("day");
  });

  it("no tiene violaciones de accesibilidad", async () => {
    const { container } = render(<AnalyticsDashboard />);
    await expectNoA11yViolations(container);
  });
});
