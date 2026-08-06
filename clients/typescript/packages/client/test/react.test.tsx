import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { createStore } from "../src/data-store";
import { defineClientPlugin, defineRoutes, defineStores } from "../src/define-plugin";
import { createAuthClient } from "../src/react";

describe("react adapter", () => {
  it("generates a hook for each registered store", () => {
    const probe = defineClientPlugin({
      id: "probe",
      routes: defineRoutes(),
      stores: () => defineStores({ probe: createStore<string>({ initial: "ready", settled: true }) }),
    });
    const auth = createAuthClient({
      baseURL: "http://localhost:8080",
      plugins: [probe],
      initialSession: null,
      crossTabSync: false,
      refetchOnWindowFocus: false,
    });

    const { result } = renderHook(() => auth.useProbe());

    expect(result.current.data).toBe("ready");
  });
});
