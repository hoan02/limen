import type { RouteContext } from "./context";
import type { AnyClientPlugin, RunRoute } from "./define-plugin";
import { chainFromDotted, pathToChain } from "./path";
import { runRoute } from "./pipeline";
import type { AnyRoute, AnyRouteDescriptor, RouteCallOptions } from "./route";
import type { StoreMap } from "./stores";
import { attachEffects } from "./stores";

export type ClientOverrides = Record<string, { basePath?: string } | undefined> | undefined;

function chainFor(plugin: AnyClientPlugin, def: AnyRoute): string[] {
  if (typeof def.as === "string") {
    return chainFromDotted(def.as);
  }
  return [...pathToChain(plugin.basePath ?? ""), ...pathToChain(def.path)];
}

function mountAtChain(target: Record<string, unknown>, pathSegments: string[], callable: unknown): void {
  let current = target;
  for (let i = 0; i < pathSegments.length - 1; i += 1) {
    const segment = pathSegments[i] as string;
    const child = current[segment];
    if (child === undefined) {
      const namespace: Record<string, unknown> = {};
      current[segment] = namespace;
      current = namespace;
    } else {
      current = child as Record<string, unknown>;
    }
  }
  const finalSegment = pathSegments[pathSegments.length - 1] as string;
  current[finalSegment] = callable;
}

function isNamespace(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value) && typeof value !== "function";
}

function mergeInto(target: Record<string, unknown>, source: Record<string, unknown>): void {
  for (const [key, value] of Object.entries(source)) {
    const existing = target[key];
    if (isNamespace(existing) && isNamespace(value)) {
      mergeInto(existing, value);
      continue;
    }
    target[key] = value;
  }
}

type BuildClientTreeArgs = {
  plugins: readonly AnyClientPlugin[];
  contexts: ReadonlyMap<AnyClientPlugin, RouteContext>;
  stores: StoreMap;
};

export function buildClientTree({ plugins, contexts, stores }: BuildClientTreeArgs): Record<string, unknown> {
  const api: Record<string, unknown> = {};

  for (const plugin of plugins) {
    const scopedCtx = contexts.get(plugin) as RouteContext;
    const contribution: Record<string, unknown> = {};

    for (const def of plugin.routes as readonly AnyRoute[]) {
      if (def.expose === false) {
        continue;
      }
      const call = (input?: unknown, opts?: RouteCallOptions) => runRoute(scopedCtx, def, input, opts);
      mountAtChain(contribution, chainFor(plugin, def), call);
    }

    if (plugin.actions !== undefined) {
      const run: RunRoute = (route, input) => runRoute(scopedCtx, route as AnyRouteDescriptor, input) as Promise<never>;
      mergeInto(contribution, plugin.actions(scopedCtx, run) as Record<string, unknown>);
    }

    if (plugin.effects !== undefined) {
      attachEffects(plugin.effects, stores, scopedCtx);
    }

    mergeInto(api, contribution);
  }

  return api;
}
