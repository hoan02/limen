import { describe, expect, it } from "vitest";
import { createAuthClient } from "../src";
import { organizationPlugin } from "../src/plugins/organization";
import { mockFetch, settle, type MockReply, type Recorded } from "./helpers";

const organizationBody = {
  id: "org-1",
  name: "Acme",
  slug: "acme",
  logo: null,
  metadata: null,
  created_at: "t0",
  updated_at: "t1",
};

const memberBody = {
  id: "mem-1",
  roles: ["admin"],
  permissions: ["member:read", "invitation:create"],
  created_at: "t0",
  updated_at: "t1",
  organization: organizationBody,
  user: { id: "u1", first_name: "Ada", last_name: "Lovelace" },
};

const signedIn = { user: { id: "u1", email: "ada@example.com", emailVerifiedAt: null } };

function setup(reply: (req: Recorded) => MockReply, me: MockReply = { body: memberBody }) {
  const { impl, calls } = mockFetch((req) =>
    req.method === "GET" && req.url.endsWith("/organizations/me") ? me : reply(req),
  );
  const auth = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins: [organizationPlugin()],
    fetchOptions: { impl },
    initialSession: signedIn,
    crossTabSync: false,
    refetchOnWindowFocus: false,
  });
  return { auth, calls };
}

describe("organization plugin", () => {
  it("reloads membership after switch and exposes the active organization", async () => {
    const { auth, calls } = setup((req) => (req.url.endsWith("/switch") ? { body: organizationBody } : { body: {} }));

    await auth.organization.switch({ id: "org-1" });
    await settle();

    expect(calls.map((call) => call.url)).toEqual([
      "http://localhost:8080/auth/organizations/switch",
      "http://localhost:8080/auth/organizations/me",
    ]);
    expect(auth.organization.active()).toMatchObject({ id: "org-1", name: "Acme", slug: "acme" });
    expect(auth.organization.activeId()).toBe("org-1");
  });

  it("clears membership without hitting /me when switching to none", async () => {
    const { auth, calls } = setup(() => ({ body: null }));

    await auth.organization.switch({ id: null });

    expect(calls.map((call) => call.url)).toEqual(["http://localhost:8080/auth/organizations/switch"]);
    expect(auth.organization.active()).toBeNull();
    expect(auth.organization.activeId()).toBeNull();
  });

  it("clears membership when the session becomes null", async () => {
    const { auth } = setup((req) =>
      req.url.endsWith("/auth/me") ? { status: 401, body: { message: "unauthorized" } } : { body: organizationBody },
    );

    await auth.organization.switch({ id: "org-1" });
    await settle();
    expect(auth.organization.activeId()).toBe("org-1");

    await auth.getSession();

    expect(auth.organization.activeId()).toBeNull();
  });

  it.each([
    { target: "org-1", keep: false },
    { target: "org-2", keep: true },
  ])("clearIf on leave $target — keep active=$keep", async ({ target, keep }) => {
    const { auth } = setup((req) => (req.url.endsWith("/leave") ? { status: 204 } : { body: organizationBody }));

    await auth.organization.switch({ id: "org-1" });
    await settle();
    expect(auth.organization.activeId()).toBe("org-1");

    await auth.organization.leave({ id: target });

    expect(auth.organization.activeId()).toBe(keep ? "org-1" : null);
  });

  it("gates custom role routes behind customRoles", () => {
    const off = setup(() => ({ body: organizationBody })).auth;
    expect((off.organization as Record<string, unknown>)["createRole"]).toBeUndefined();

    const { impl } = mockFetch(() => ({ body: organizationBody }));
    const on = createAuthClient({
      baseURL: "http://localhost:8080",
      plugins: [organizationPlugin({ customRoles: true })],
      fetchOptions: { impl },
      crossTabSync: false,
      refetchOnWindowFocus: false,
    });
    expect(typeof on.organization.createRole).toBe("function");
  });
});
