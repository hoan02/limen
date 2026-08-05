import { onNotify } from "nanostores";
import { createBroadcastChannel } from "./broadcast-channel";
import { deepJsonEqual } from "./json-deep-equal";
import type { SessionStore } from "./session-store";
import type { Session } from "./types";

const CHANNEL_NAME = "limen.session";
const FOCUS_REFETCH_THROTTLE_MS = 5_000;

type SyncMessage<TFields> = { data: Session<TFields> | null };

type SessionSyncOptions = {
  /** Mirror session changes to other same-origin tabs. */
  crossTabSync: boolean;
  /** Re-validate against `/me` when the tab returns to the foreground. */
  refetchOnWindowFocus: boolean;
};

export function createSessionSync<TFields = unknown>(
  store: SessionStore<TFields>,
  options: SessionSyncOptions,
): () => void {
  const teardowns: Array<() => void> = [];
  if (options.crossTabSync) {
    teardowns.push(syncAcrossTabs(store));
  }

  if (options.refetchOnWindowFocus) {
    teardowns.push(refetchOnFocus(store));
  }

  return () => {
    for (const teardown of teardowns) {
      teardown();
    }
  };
}

function syncAcrossTabs<TFields>(store: SessionStore<TFields>): () => void {
  const port = createBroadcastChannel<SyncMessage<TFields>>(CHANNEL_NAME);
  let lastData = store.$state.get().data;

  const unsubscribe = port.subscribe((message) => {
    // Mark remote updates as seen before applying them to avoid echoing them.
    lastData = message.data;
    store.setData(message.data);
  });

  const unbindNotify = onNotify(store.$state, () => {
    const data = store.$state.get().data;
    if (deepJsonEqual(data, lastData)) {
      return;
    }
    lastData = data;
    port.post({ data: data });
  });

  return () => {
    unbindNotify();
    unsubscribe();
    port.close();
  };
}

function refetchOnFocus<TFields>(store: SessionStore<TFields>): () => void {
  if (typeof document === "undefined") {
    return () => {};
  }
  const onVisibilityChange = (): void => {
    if (document.visibilityState === "visible") {
      void store.refetch({ maxAgeMs: FOCUS_REFETCH_THROTTLE_MS, skipWhenEmpty: true });
    }
  };
  document.addEventListener("visibilitychange", onVisibilityChange);
  return () => document.removeEventListener("visibilitychange", onVisibilityChange);
}
