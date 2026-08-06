import type { ReadableAtom } from "nanostores";
import { onNotify } from "nanostores";
import type { AnyRouteContext, RouteContext } from "./context";
import type { DataStore, StoreState } from "./data-store";
import { capitalize } from "./helpers";
import { deepJsonEqual } from "./json-deep-equal";

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- store values vary per plugin
export type PluginStores = Readonly<Record<string, DataStore<any>>>;

export type StoreMap = PluginStores;

declare const REF_VALUE: unique symbol;

/** Published store handle; apps see `$name`, other plugins resolve via {@link StoreRegistry}. */
export type StoreRef<Value> = {
  readonly name: string;
  readonly [REF_VALUE]?: Value;
};

export function storeRef<Value>(name: string): StoreRef<Value> {
  return { name };
}

export type StoreRegistry = {
  get<Value>(ref: StoreRef<Value>): DataStore<Value> | undefined;
};

export function createStoreRegistry(stores: () => StoreMap): StoreRegistry {
  return {
    get: <Value>(ref: StoreRef<Value>) => stores()[ref.name] as DataStore<Value> | undefined,
  };
}

export type EffectTrigger = "data" | "write";

export type StoreEffect = {
  readonly ref: StoreRef<unknown>;
  readonly run: (state: never, ctx: AnyRouteContext) => void;
  readonly when: EffectTrigger;
};

/** Defaults to `when: "data"` so pending/error flips do not re-run the effect. */
export function effect<Value>(
  ref: StoreRef<Value>,
  run: (state: StoreState<Value>, ctx: RouteContext) => void,
  options?: { when?: EffectTrigger },
): StoreEffect {
  return { ref: ref as StoreRef<unknown>, run: run as StoreEffect["run"], when: options?.when ?? "data" };
}

export function currentData<Value>(ctx: AnyRouteContext, ref: StoreRef<Value>): Value | undefined {
  return ctx.stores.get(ref)?.current().data;
}

type StoreOwner = {
  readonly id: string;
  readonly stores?: (ctx: AnyRouteContext) => PluginStores;
};

export function collectStores(
  plugins: readonly StoreOwner[],
  contexts: ReadonlyMap<StoreOwner, AnyRouteContext>,
  core: PluginStores,
): StoreMap {
  const stores: Record<string, DataStore<unknown>> = { ...core };

  for (const plugin of plugins) {
    const ctx = contexts.get(plugin);
    if (plugin.stores === undefined || ctx === undefined) {
      continue;
    }
    for (const [name, store] of Object.entries(plugin.stores(ctx))) {
      if (name in stores) {
        throw new Error(`limen: plugin "${plugin.id}" registers store "${name}", which is already registered`);
      }
      stores[name] = store;
    }
  }

  return stores;
}

export function attachEffects(effects: readonly StoreEffect[], stores: StoreMap, ctx: AnyRouteContext): void {
  for (const { ref, run, when } of effects) {
    const store = stores[ref.name];
    if (store === undefined) {
      continue;
    }

    if (when === "write") {
      onNotify(store.$state, () => runEffect(run, store.current(), ctx));
      continue;
    }

    let previous = store.current().data;
    onNotify(store.$state, () => {
      const state = store.current();
      if (deepJsonEqual(state.data, previous)) {
        return;
      }
      previous = state.data;
      runEffect(run, state, ctx);
    });
  }
}

function runEffect(run: StoreEffect["run"], value: unknown, ctx: AnyRouteContext): void {
  try {
    (run as (value: unknown, ctx: AnyRouteContext) => void)(value, ctx);
  } catch (error) {
    console.error("limen: store effect failed", error);
  }
}

export function storeAtoms(stores: StoreMap): Record<string, unknown> {
  const atoms: Record<string, unknown> = {};
  for (const [name, store] of Object.entries(stores)) {
    atoms[`$${name}`] = store.$state;
  }
  return atoms;
}

export function attachStoreHooks(client: object, wrap: (store: ReadableAtom<unknown>) => unknown): void {
  for (const [key, store] of Object.entries(client)) {
    if (!key.startsWith("$")) {
      continue;
    }
    Object.assign(client, { [`use${capitalize(key.slice(1))}`]: () => wrap(store as ReadableAtom<unknown>) });
  }
}
