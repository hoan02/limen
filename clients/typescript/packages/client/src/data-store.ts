import { atom, onMount, type ReadableAtom } from "nanostores";
import { LimenError } from "./errors";

export type StoreState<T> = {
  data: T;
  /** Writes do not set it — only loads. */
  isPending: boolean;
  /** True once `data` is known — seeded by the app, or loaded successfully. */
  settled: boolean;
  /** The last load failure, cleared on the next load or write. */
  error: LimenError | null;
};

export type StoreLoader<T> = () => Promise<T>;

export type RefetchOptions = {
  /** Skip the load when the last one ran within this many milliseconds. */
  maxAgeMs?: number;
  /** Skip the load while `data` is `null`. */
  skipWhenEmpty?: boolean;
  /** Start a load even when one is in flight: joining it would resolve with the state the write replaced. */
  force?: boolean;
};

export type DataStore<T> = {
  /** Subscribe or `get()` for UI — both mount the store, starting `fetchOnMount`. */
  readonly $state: ReadableAtom<StoreState<T>>;
  /** Snapshot without mounting — safe inside effects and non-UI reads. */
  current(): StoreState<T>;
  setData(data: T): void;
  refetch(options?: RefetchOptions): Promise<void>;
};

type CreateStoreOptions<T> = {
  initial: T;
  /** Whether `initial` is a known value rather than a placeholder. */
  settled?: boolean;
  loader?: StoreLoader<T>;
  fetchOnMount?: boolean;
  onMount?: (store: DataStore<T>) => (() => void) | void;
};

function asLimenError(error: unknown): LimenError {
  if (error instanceof LimenError) {
    return error;
  }
  return new LimenError(error instanceof Error ? error.message : "Failed to load", 0, "unknown");
}

export function createStore<T>(options: CreateStoreOptions<T>): DataStore<T> {
  const $state = atom<StoreState<T>>({
    data: options.initial,
    isPending: false,
    settled: options.settled === true,
    error: null,
  });

  let inFlight: Promise<void> | null = null;
  let lastLoadedAt = 0;
  // Bumped on every write so an older load cannot overwrite newer state.
  let writeVersion = 0;
  const isStale = (version: number): boolean => version !== writeVersion;
  const current = (): StoreState<T> => $state.value as StoreState<T>;

  const load = async (loader: StoreLoader<T>): Promise<void> => {
    const version = ++writeVersion;
    $state.set({ ...current(), isPending: true, error: null });
    try {
      const data = await loader();
      if (isStale(version)) {
        return;
      }
      $state.set({ data, isPending: false, settled: true, error: null });
    } catch (error) {
      if (isStale(version)) {
        return;
      }
      // `settled` is left alone: after a failure the real value is still unknown.
      $state.set({ ...current(), isPending: false, error: asLimenError(error) });
    }
  };

  const setData = (data: T): void => {
    writeVersion++;
    $state.set({ data, isPending: false, settled: true, error: null });
  };

  const refetch = (refetchOptions?: RefetchOptions): Promise<void> => {
    const { loader } = options;
    if (loader === undefined) {
      return Promise.resolve();
    }

    const { maxAgeMs, skipWhenEmpty, force } = refetchOptions ?? {};
    if (skipWhenEmpty === true && current().data === null) {
      return Promise.resolve();
    }
    if (maxAgeMs !== undefined && Date.now() - lastLoadedAt < maxAgeMs) {
      return Promise.resolve();
    }

    if (inFlight === null || force === true) {
      const started = load(loader).finally(() => {
        // Only clear the slot if a forced load has not already replaced it.
        if (inFlight === started) {
          inFlight = null;
          lastLoadedAt = Date.now();
        }
      });
      inFlight = started;
    }
    return inFlight;
  };

  const store: DataStore<T> = { $state, current, setData, refetch };

  onMount($state, () => {
    if (options.fetchOnMount === true) {
      void refetch();
    }
    return options.onMount?.(store);
  });

  return store;
}
