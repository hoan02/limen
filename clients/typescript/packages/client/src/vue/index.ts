import { createAuthClient as createCoreClient } from "../client";
import type { AnyClientPlugin } from "../define-plugin";
import type { StoresOf } from "../infer";
import { attachStoreHooks } from "../stores";
import type { Prettify } from "../type-utils";
import type { AuthClient, CreateAuthClientOptions } from "../types";
import type { ReactiveValue } from "./vue-store";
import { useStore } from "./vue-store";

/** One composable per registered store, each returning a readonly ref. */
export type VueStoreHooks<Stores> = {
  readonly [K in keyof Stores & string as `use${Capitalize<K>}`]: () => ReactiveValue<Stores[K]>;
};

export type VueAuthClient<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = Prettify<
  AuthClient<Plugins, TFields> & VueStoreHooks<StoresOf<Plugins, TFields>>
>;

export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): VueAuthClient<Plugins, TFields> {
  const client = createCoreClient<Plugins, TFields>(opts);

  attachStoreHooks(client, (store) => useStore(store));

  return client as VueAuthClient<Plugins, TFields>;
}

export type { SessionState, SessionStore } from "../session-store";
export type { AuthClient, CreateAuthClientOptions, Session, User } from "../types";
export { useStore } from "./vue-store";
export type { ReactiveValue } from "./vue-store";
