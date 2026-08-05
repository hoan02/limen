import type { Readable } from "svelte/store";
import { createAuthClient as createCoreClient } from "../client";
import type { AnyClientPlugin } from "../define-plugin";
import type { StoresOf } from "../infer";
import { attachStoreHooks } from "../stores";
import type { Prettify } from "../type-utils";
import type { AuthClient, CreateAuthClientOptions } from "../types";

/** One accessor per registered store, each returning a Svelte readable. */
export type SvelteStoreHooks<Stores> = {
  readonly [K in keyof Stores & string as `use${Capitalize<K>}`]: () => Readable<Stores[K]>;
};

export type SvelteAuthClient<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = Prettify<
  AuthClient<Plugins, TFields> & SvelteStoreHooks<StoresOf<Plugins, TFields>>
>;

export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): SvelteAuthClient<Plugins, TFields> {
  const client = createCoreClient<Plugins, TFields>(opts);

  // Nanostores atoms already satisfy the Svelte store contract.
  attachStoreHooks(client, (store) => store);

  return client as SvelteAuthClient<Plugins, TFields>;
}

export type { SessionState, SessionStore } from "../session-store";
export type { AuthClient, CreateAuthClientOptions, Session, User } from "../types";
