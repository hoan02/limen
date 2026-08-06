import { describe, expect, it, vi } from "vitest";
import { createAuthStore } from "../src/auth-store";
import type { AnyRouteContext } from "../src/context";
import { LimenError } from "../src/errors";
import type { SessionStore } from "../src/session-store";
import type { Session } from "../src/types";

const tick = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 0));

function fakeCtx(options: {
  session: Session | null;
  settled?: boolean;
  fetch?: AnyRouteContext["fetch"];
}): AnyRouteContext {
  const fetch: AnyRouteContext["fetch"] =
    options.fetch ??
    (async () => {
      throw new Error("unexpected fetch");
    });

  return {
    fetch,
    redirect: () => false,
    parseSession: (raw) => raw as Session,
    setSession: () => {},
    refetchSession: async () => {},
    currentSession: () => options.session,
    store: {
      current: () => ({
        data: options.session,
        isPending: false,
        settled: options.settled ?? true,
        error: null,
      }),
    } as SessionStore,
    stores: { get: () => undefined },
  };
}

describe("createAuthStore", () => {
  it("loads and parses on mount", async () => {
    const fetch = vi.fn(async () => ({ name: "Acme" })) as unknown as AnyRouteContext["fetch"];
    const store = createAuthStore({
      ctx: fakeCtx({ session: { user: { id: "u1", email: "a@b.c", emailVerifiedAt: null } }, fetch }),
      path: "/active",
      parse: (raw) => ({ ...(raw as { name: string }), ok: true }),
    });

    const unsubscribe = store.$state.listen(() => {});
    await tick();

    expect(fetch).toHaveBeenCalledWith("/active", { method: "GET" });
    expect(store.current().data).toEqual({ name: "Acme", ok: true });
    unsubscribe();
  });

  it("skips the request when the session is settled null", async () => {
    const fetch = vi.fn(async () => ({ name: "Acme" })) as unknown as AnyRouteContext["fetch"];
    const store = createAuthStore({
      ctx: fakeCtx({ session: null, settled: true, fetch }),
      path: "/active",
    });

    const unsubscribe = store.$state.listen(() => {});
    await tick();

    expect(fetch).not.toHaveBeenCalled();
    expect(store.current().data).toBeNull();
    unsubscribe();
  });

  it("treats 401 as null", async () => {
    const fetch = vi.fn(async () => {
      throw new LimenError("nope", 401);
    }) as unknown as AnyRouteContext["fetch"];
    const store = createAuthStore({
      ctx: fakeCtx({ session: { user: { id: "u1", email: "a@b.c", emailVerifiedAt: null } }, fetch }),
      path: "/active",
    });

    const unsubscribe = store.$state.listen(() => {});
    await tick();

    expect(store.current().data).toBeNull();
    expect(store.current().error).toBeNull();
    unsubscribe();
  });

  it("surfaces non-401 failures", async () => {
    const fetch = vi.fn(async () => {
      throw new LimenError("boom", 500);
    }) as unknown as AnyRouteContext["fetch"];
    const store = createAuthStore({
      ctx: fakeCtx({ session: { user: { id: "u1", email: "a@b.c", emailVerifiedAt: null } }, fetch }),
      path: "/active",
    });

    const unsubscribe = store.$state.listen(() => {});
    await tick();

    expect(store.current().data).toBeNull();
    expect(store.current().error?.status).toBe(500);
    unsubscribe();
  });

  it("forwards fetch options", async () => {
    const fetch = vi.fn(async () => ({ ok: true })) as unknown as AnyRouteContext["fetch"];
    const store = createAuthStore({
      ctx: fakeCtx({ session: { user: { id: "u1", email: "a@b.c", emailVerifiedAt: null } }, fetch }),
      path: "/items",
      fetch: {
        method: "POST",
        query: { page: "1" },
        headers: { "X-Test": "1" },
        body: { filter: "active" },
        timeout: 5_000,
      },
    });

    const unsubscribe = store.$state.listen(() => {});
    await tick();

    expect(fetch).toHaveBeenCalledWith("/items", {
      method: "POST",
      query: { page: "1" },
      headers: { "X-Test": "1" },
      body: { filter: "active" },
      timeout: 5_000,
    });
    unsubscribe();
  });
});
