import { createAuthClient as createCoreClient } from "../client";
import type { AnyClientPlugin } from "../define-plugin";
import type { StoresOf } from "../infer";
import { attachStoreHooks } from "../stores";
import type { Prettify } from "../type-utils";
import type { AuthClient, CreateAuthClientOptions } from "../types";
import { useStore } from "./react-store";

/** One hook per registered store, named after it: `session` becomes `useSession()`. */
export type ReactStoreHooks<Stores> = {
  readonly [K in keyof Stores & string as `use${Capitalize<K>}`]: () => Stores[K];
};

export type ReactAuthClient<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = Prettify<
  AuthClient<Plugins, TFields> & ReactStoreHooks<StoresOf<Plugins, TFields>>
>;

export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): ReactAuthClient<Plugins, TFields> {
  const client = createCoreClient<Plugins, TFields>(opts);

  attachStoreHooks(client, (store) => useStore(store));

  return client as ReactAuthClient<Plugins, TFields>;
}

export type { SessionState, SessionStore } from "../session-store";
export type { AuthClient, CreateAuthClientOptions, Session, User } from "../types";
export { useStore } from "./react-store";
