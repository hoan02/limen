import { describe, expect, it, vi } from "vitest";
import { createStore } from "../src/data-store";
import { LimenError } from "../src/errors";

const tick = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 0));

describe("createStore — current vs mount", () => {
  it("current() does not mount or start fetchOnMount", async () => {
    const loader = vi.fn(async () => "loaded");
    const store = createStore<string | null>({ initial: null, fetchOnMount: true, loader });

    expect(store.current().data).toBeNull();
    await tick();

    expect(loader).not.toHaveBeenCalled();
  });

  it("subscribing to $state mounts and runs fetchOnMount", async () => {
    const loader = vi.fn(async () => "loaded");
    const store = createStore<string | null>({ initial: null, fetchOnMount: true, loader });

    const unsubscribe = store.$state.listen(() => {});
    await tick();

    expect(loader).toHaveBeenCalledTimes(1);
    expect(store.current().data).toBe("loaded");
    unsubscribe();
  });
});

describe("createStore — refetch", () => {
  it("joins an in-flight load when force is not set", async () => {
    let resolveLoad!: (value: string) => void;
    const loader = vi.fn(
      () =>
        new Promise<string>((resolve) => {
          resolveLoad = resolve;
        }),
    );
    const store = createStore<string | null>({ initial: null, loader });

    const first = store.refetch();
    const second = store.refetch();
    expect(loader).toHaveBeenCalledTimes(1);

    resolveLoad("a");
    await Promise.all([first, second]);

    expect(store.current().data).toBe("a");
  });

  it("starts a second load when force is true", async () => {
    const resolvers: Array<(value: string) => void> = [];
    const loader = vi.fn(
      () =>
        new Promise<string>((resolve) => {
          resolvers.push(resolve);
        }),
    );
    const store = createStore<string | null>({ initial: null, loader });

    const first = store.refetch();
    const second = store.refetch({ force: true });
    expect(loader).toHaveBeenCalledTimes(2);

    resolvers[0]?.("stale");
    resolvers[1]?.("fresh");
    await Promise.all([first, second]);

    expect(store.current().data).toBe("fresh");
  });

  it("setData makes an older in-flight load lose", async () => {
    let resolveLoad!: (value: string) => void;
    const store = createStore<string | null>({
      initial: null,
      loader: () =>
        new Promise<string>((resolve) => {
          resolveLoad = resolve;
        }),
    });

    const pending = store.refetch();
    store.setData("written");
    resolveLoad("from-loader");
    await pending;

    expect(store.current().data).toBe("written");
    expect(store.current().isPending).toBe(false);
  });

  it("keeps settled false after a load failure when never settled", async () => {
    const store = createStore<string | null>({
      initial: null,
      loader: async () => {
        throw new LimenError("boom", 500);
      },
    });

    await store.refetch();

    expect(store.current()).toMatchObject({
      data: null,
      settled: false,
      isPending: false,
    });
    expect(store.current().error?.status).toBe(500);
  });

  it("skips when skipWhenEmpty and data is null", async () => {
    const loader = vi.fn(async () => "x");
    const store = createStore<string | null>({ initial: null, loader });

    await store.refetch({ skipWhenEmpty: true });

    expect(loader).not.toHaveBeenCalled();
  });

  it("skips when maxAgeMs has not elapsed since the last load", async () => {
    vi.useFakeTimers();
    try {
      const loader = vi.fn(async () => "x");
      const store = createStore<string | null>({ initial: null, loader });

      await store.refetch();
      await store.refetch({ maxAgeMs: 5_000 });
      expect(loader).toHaveBeenCalledTimes(1);

      vi.advanceTimersByTime(5_001);
      await store.refetch({ maxAgeMs: 5_000 });
      expect(loader).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });
});
