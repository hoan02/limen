import { describe, expect, it, vi } from "vitest";
import { LimenError } from "../src/errors";
import type { SessionStore } from "../src/session-store";
import { createSessionStore } from "../src/session-store";
import type { Session } from "../src/types";

/** The session loader fetches `/me` and parses it; tests stand both in. */
function withHydrator(
  hydrator: () => Promise<Session>,
  options: { initialSession?: Session | null } = {},
): SessionStore {
  return createSessionStore({
    fetch: hydrator as <T>() => Promise<T>,
    parseSession: (raw) => raw as Session,
    ...options,
  });
}

function user(id: string): Session {
  return { user: { id, email: `${id}@example.com`, emailVerifiedAt: null } };
}

function tick(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

describe("createSessionStore", () => {
  it("does not hydrate when seeded with initialSession", async () => {
    const hydrator = vi.fn(async () => user("user-1"));
    const store = withHydrator(hydrator, { initialSession: user("seed") });

    const unsubscribe = store.$state.listen(() => {});
    await tick();

    expect(hydrator).not.toHaveBeenCalled();
    expect(store.$state.get().data?.user.id).toBe("seed");
    unsubscribe();
  });

  it("treats a 401 as signed out", async () => {
    const store = withHydrator(
      async () => {
        throw new LimenError("nope", 401);
      },
      { initialSession: user("user-1") },
    );

    await store.refetch();

    expect(store.$state.get().data).toBeNull();
    expect(store.$state.get().error).toBeNull();
  });
});
