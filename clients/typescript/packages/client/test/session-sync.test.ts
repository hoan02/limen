import { atom } from "nanostores";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { SessionState } from "../src/session-store";
import { createSessionSync } from "../src/session-sync";
import type { Session } from "../src/types";

const STORAGE_KEY = "limen.session-sync";

function user(id: string, fields: Partial<Session["user"]> = {}): Session {
  return { user: { id, email: `${id}@example.com`, emailVerifiedAt: null, ...fields } };
}

function stateOf(session: Session | null): SessionState {
  return { data: session, isPending: false, settled: true, error: null };
}

function tick(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

class FakeBroadcastChannel {
  static peers = new Map<string, Set<FakeBroadcastChannel>>();
  static reset(): void {
    FakeBroadcastChannel.peers.clear();
  }

  onmessage: ((event: MessageEvent) => void) | null = null;
  private closed = false;

  constructor(public readonly name: string) {
    const set = FakeBroadcastChannel.peers.get(name) ?? new Set();
    set.add(this);
    FakeBroadcastChannel.peers.set(name, set);
  }

  postMessage(data: unknown): void {
    for (const peer of FakeBroadcastChannel.peers.get(this.name) ?? []) {
      if (peer === this || peer.closed) {
        continue;
      }
      peer.onmessage?.({ data: structuredClone(data) } as MessageEvent);
    }
  }

  close(): void {
    this.closed = true;
    FakeBroadcastChannel.peers.get(this.name)?.delete(this);
  }
}

function makeStore(initial: Session | null = null) {
  const $state = atom<SessionState>(stateOf(initial));
  const setData = vi.fn((session: Session | null) => {
    $state.set(stateOf(session));
  });
  const refetch = vi.fn(async () => {});
  const current = () => $state.get();
  return { $state, current, setData, refetch };
}

type SyncTab = ReturnType<typeof makeStore>;

function syncCrossTab(tab: SyncTab): () => void {
  return createSessionSync(tab, { crossTabSync: true, refetchOnWindowFocus: false });
}

describe("createSessionSync cross-tab", () => {
  beforeEach(() => {
    FakeBroadcastChannel.reset();
    vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("applies a data snapshot to other tabs without refetch", async () => {
    const tabA = makeStore();
    const tabB = makeStore();
    syncCrossTab(tabA);
    syncCrossTab(tabB);

    tabA.$state.set(stateOf(user("user-1", { emailVerifiedAt: "2026-06-18T00:00:00Z" })));
    await tick();

    expect(tabB.setData).toHaveBeenCalledTimes(1);
    expect(tabB.$state.get().data).toMatchObject({
      user: { id: "user-1", emailVerifiedAt: "2026-06-18T00:00:00Z" },
    });
    expect(tabB.refetch).not.toHaveBeenCalled();
    expect(tabA.setData).not.toHaveBeenCalled();
  });

  it("does not re-broadcast a remotely applied change", async () => {
    const tabA = makeStore();
    const tabB = makeStore();
    const tabC = makeStore();
    syncCrossTab(tabA);
    syncCrossTab(tabB);
    syncCrossTab(tabC);

    tabA.$state.set(stateOf(user("user-1")));
    await tick();

    expect(tabB.setData).toHaveBeenCalledTimes(1);
    expect(tabC.setData).toHaveBeenCalledTimes(1);
    expect(tabA.setData).not.toHaveBeenCalled();
  });

  it("does not broadcast when only isPending or error change", async () => {
    const tabA = makeStore(user("user-1"));
    const tabB = makeStore(user("user-1"));
    syncCrossTab(tabA);
    syncCrossTab(tabB);

    tabA.$state.set({ data: user("user-1"), isPending: true, settled: true, error: null });
    await tick();

    expect(tabB.setData).not.toHaveBeenCalled();
  });

  it("stops syncing after its dispose is called", async () => {
    const tabA = makeStore();
    const tabB = makeStore();
    syncCrossTab(tabA);
    const disposeB = syncCrossTab(tabB);

    disposeB();
    tabA.$state.set(stateOf(user("user-1")));
    await tick();

    expect(tabB.setData).not.toHaveBeenCalled();
  });
});

describe("createSessionSync storage fallback", () => {
  beforeEach(() => {
    vi.stubGlobal("BroadcastChannel", undefined);
    globalThis.localStorage.clear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    globalThis.localStorage.clear();
  });

  it("applies a snapshot from a storage event to the session store", async () => {
    const tab = makeStore();
    syncCrossTab(tab);

    window.dispatchEvent(
      new StorageEvent("storage", { key: STORAGE_KEY, newValue: JSON.stringify({ data: user("user-1") }) }),
    );
    await tick();

    expect(tab.setData).toHaveBeenCalledTimes(1);
    expect(tab.$state.get().data?.user.id).toBe("user-1");
  });
});

describe("createSessionSync refetchOnWindowFocus", () => {
  function setVisibility(value: "visible" | "hidden"): void {
    Object.defineProperty(document, "visibilityState", { configurable: true, value });
  }

  afterEach(() => {
    setVisibility("visible");
  });

  it("refetches with a throttle window when the tab becomes visible", async () => {
    const tab = makeStore(user("user-1"));
    createSessionSync(tab, { crossTabSync: false, refetchOnWindowFocus: true });

    setVisibility("visible");
    document.dispatchEvent(new Event("visibilitychange"));
    await tick();

    expect(tab.refetch).toHaveBeenCalledWith({ maxAgeMs: expect.any(Number), skipWhenEmpty: true });
  });

  it("does not refetch while hidden", async () => {
    const tab = makeStore(user("user-1"));
    createSessionSync(tab, { crossTabSync: false, refetchOnWindowFocus: true });

    setVisibility("hidden");
    document.dispatchEvent(new Event("visibilitychange"));
    await tick();

    expect(tab.refetch).not.toHaveBeenCalled();
  });

  it("removes the focus listener on dispose", async () => {
    const tab = makeStore(user("user-1"));
    const dispose = createSessionSync(tab, { crossTabSync: false, refetchOnWindowFocus: true });
    dispose();

    setVisibility("visible");
    document.dispatchEvent(new Event("visibilitychange"));
    await tick();

    expect(tab.refetch).not.toHaveBeenCalled();
  });
});
