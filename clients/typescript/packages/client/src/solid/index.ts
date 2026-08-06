import type { Accessor } from "solid-js";
import { createAuthClient as createCoreClient } from "../client";
import type { AnyClientPlugin } from "../define-plugin";
import type { StoresOf } from "../infer";
import { attachStoreHooks } from "../stores";
import type { Prettify } from "../type-utils";
import type { AuthClient, CreateAuthClientOptions } from "../types";
import { useStore } from "./solid-store";

/** One primitive per registered store, each returning an accessor. */
export type SolidStoreHooks<Stores> = {
  readonly [K in keyof Stores & string as `use${Capitalize<K>}`]: () => Accessor<Stores[K]>;
};

export type SolidAuthClient<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = Prettify<
  AuthClient<Plugins, TFields> & SolidStoreHooks<StoresOf<Plugins, TFields>>
>;

export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): SolidAuthClient<Plugins, TFields> {
  const client = createCoreClient<Plugins, TFields>(opts);

  attachStoreHooks(client, (store) => useStore(store));

  return client as SolidAuthClient<Plugins, TFields>;
}

export type { SessionState, SessionStore } from "../session-store";
export type { AuthClient, CreateAuthClientOptions, Session, User } from "../types";
export { useStore } from "./solid-store";
