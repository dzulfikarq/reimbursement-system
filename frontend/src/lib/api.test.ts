// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Handler receives nothing; returns the mocked upstream response.
type MockHandler = { status: number; body?: unknown };
type HandlerFn = () => MockHandler;

function makeHarness() {
  const calls: { method: string; url: string; csrf?: string }[] = [];
  const handlers: Record<string, HandlerFn> = {};
  let api: any;

  const adapter = async (config: any) => {
    const method = String(config.method ?? "get").toLowerCase();
    const url = String(config.url ?? "");
    // config.headers is an AxiosHeaders instance (has its own .get()).
    const csrf = config.headers?.get?.("X-CSRF-Token") ?? undefined;
    calls.push({ method, url, csrf });

    const fn = handlers[`${method} ${url}`];
    if (!fn) throw new Error(`no mock for ${method} ${url}`);
    const { status, body } = fn();
    const response = { data: body ?? {}, status, statusText: String(status), headers: {}, config };
    if (status >= 200 && status < 300) return response;
    const err: any = new Error("Request failed");
    err.response = response;
    err.config = config;
    err.isAxiosError = true;
    throw err;
  };

  async function start() {
    vi.resetModules(); // fresh module-level refreshPromise per test
    const mod = await import("./api");
    api = mod.api;
    api.defaults.adapter = adapter;
    return api;
  }

  return {
    calls,
    on(method: string, url: string, fn: HandlerFn) {
      handlers[`${method} ${url}`] = fn;
    },
    start,
  };
}

describe("lib/api interceptors (real instance)", () => {
  let harness: ReturnType<typeof makeHarness>;

  beforeEach(() => {
    document.cookie = "";
    harness = makeHarness();
  });
  afterEach(() => {
    document.cookie = "";
  });

  it("sends X-CSRF-Token from cookie on mutations only", async () => {
    document.cookie = "csrf_token=tok-123";
    const api = await harness.start();
    harness.on("get", "/claims", () => ({ status: 200 }));
    harness.on("post", "/claims", () => ({ status: 201 }));

    await api.get("/claims");
    await api.post("/claims", {});

    expect(harness.calls.find((c) => c.method === "get")?.csrf).toBeUndefined();
    expect(harness.calls.find((c) => c.method === "post")?.csrf).toBe("tok-123");
  });

  it("single-flights one refresh for concurrent 401s and replays originals", async () => {
    let refreshed = false;
    harness.on("post", "/auth/refresh", () => {
      refreshed = true;
      return { status: 200 };
    });
    harness.on("get", "/protected", () =>
      refreshed ? { status: 200 } : { status: 401 },
    );
    const api = await harness.start();

    // Concurrent requests share one refresh; all replay to 200.
    const [r1, r2, r3] = await Promise.all([
      api.get("/protected"),
      api.get("/protected"),
      api.get("/protected"),
    ]);

    expect(r1.status).toBe(200);
    expect(r2.status).toBe(200);
    expect(r3.status).toBe(200);
    expect(harness.calls.filter((c) => c.url === "/auth/refresh").length).toBe(1);
    expect(harness.calls.filter((c) => c.url === "/protected").length).toBe(6); // 3×(401 + replay)
  });

  it("does not loop when refresh itself fails", async () => {
    harness.on("get", "/protected", () => ({ status: 401 }));
    harness.on("post", "/auth/refresh", () => ({ status: 401 }));
    const api = await harness.start();

    await expect(api.get("/protected")).rejects.toMatchObject({
      response: { status: 401 },
    });

    expect(harness.calls.filter((c) => c.url === "/auth/refresh").length).toBe(1);
    expect(harness.calls.filter((c) => c.url === "/protected").length).toBe(1);
  });
});
