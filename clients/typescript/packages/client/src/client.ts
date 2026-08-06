import type { ClientOverrides } from "./build-tree";
import { buildClientTree } from "./build-tree";
import { DEFAULT_ENVELOPE_CONFIG } from "./constants";
import type { AnyRouteContext, RouteContext } from "./context";
import type { AnyClientPlugin } from "./define-plugin";
import { Fetcher } from "./fetcher";
import { ensureLeadingSlash, kebabToCamel, normalizeBasePath, stripTrailingSlash } from "./helpers";
import { HookRunner } from "./hooks";
import { defaultSessionParse } from "./normalize";
import type { FetchInit } from "./plugin";
import { coreClientPlugin } from "./routes";
import { createSessionStore } from "./session-store";
import type { StoreMap } from "./stores";
import { collectStores, createStoreRegistry, storeAtoms } from "./stores";
import type { AuthClient, ClientFetchOptions, CreateAuthClientOptions, EnvelopeConfig, RedirectFn } from "./types";

export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): AuthClient<Plugins, TFields> {
  const baseURL = stripTrailingSlash(opts.baseURL);
  const basePath = normalizeBasePath(opts.basePath ?? "/auth");

  const userPlugins = (opts.plugins ?? []) as readonly AnyClientPlugin[];
  const plugins: readonly AnyClientPlugin[] = [coreClientPlugin<TFields>(), ...userPlugins];

  const hooks = new HookRunner(plugins);
  const envelope = { ...DEFAULT_ENVELOPE_CONFIG, ...opts.envelope } satisfies EnvelopeConfig;
  const fetcher = buildFetcher(baseURL, basePath, envelope, hooks, opts.fetchOptions ?? {});

  const parseSession = opts.parseSession ?? defaultSessionParse;
  const redirect = resolveRedirect(opts.redirectFn);
  const baseFetch = <T>(path: string, init?: FetchInit) => fetcher.fetch<T>(path, init);

  const store = createSessionStore<TFields>({
    fetch: baseFetch,
    parseSession,
    crossTabSync: opts.crossTabSync !== false,
    refetchOnWindowFocus: opts.refetchOnWindowFocus !== false,
    ...(opts.initialSession !== undefined ? { initialSession: opts.initialSession } : {}),
  });

  let stores: StoreMap = {};

  const ctx: RouteContext<TFields> = {
    fetch: baseFetch,
    redirect,
    parseSession,
    setSession: (session) => store.setData(session),
    refetchSession: () => store.refetch(),
    currentSession: () => store.$state.get().data,
    store,
    stores: createStoreRegistry(() => stores),
  };

  const overrides = opts.overrides as ClientOverrides;
  const contexts = new Map(plugins.map((plugin) => [plugin, scopeContext(ctx, fetcher, plugin, overrides)]));

  stores = collectStores(plugins, contexts, { session: store });

  const api = buildClientTree({ plugins, contexts, stores });

  const client: Record<string, unknown> = {
    baseURL,
    basePath,
    ...api,
    ...storeAtoms(stores),
  };

  return client as AuthClient<Plugins, TFields>;
}

function buildFetcher(
  baseURL: string,
  basePath: string,
  envelope: EnvelopeConfig,
  hooks: HookRunner,
  fetchOptions: ClientFetchOptions,
): Fetcher {
  return new Fetcher({
    baseURL,
    basePath,
    envelope,
    hooks,
    fetchOptions,
  });
}

function resolveRedirect(redirect: RedirectFn | undefined): RedirectFn {
  return (url: string) => {
    if (redirect !== undefined) {
      return redirect(url);
    }
    if (typeof window !== "undefined" && typeof window.location !== "undefined") {
      window.location.href = url;
      return true;
    }
    return false;
  };
}

/** Scope a context's `fetch` to one plugin's base path (after client `overrides`). */
function scopeContext(
  ctx: AnyRouteContext,
  fetcher: Fetcher,
  plugin: AnyClientPlugin,
  overrides: ClientOverrides,
): RouteContext {
  const defaultBase = normalizeBasePath(plugin.basePath ?? "");
  const overrideBase = overrides?.[kebabToCamel(plugin.id)]?.basePath;
  const resolvedBase = normalizeBasePath(overrideBase ?? plugin.basePath ?? "");
  return {
    ...ctx,
    fetch: <T>(path: string, init?: FetchInit) => {
      const absolute = init?.absolute === true;
      const requestPath = (absolute ? "" : resolvedBase) + ensureLeadingSlash(path);
      const routePath = (absolute ? "" : defaultBase) + ensureLeadingSlash(path);
      return fetcher.fetch<T>(requestPath, init, routePath);
    },
  };
}
