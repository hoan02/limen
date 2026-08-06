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

  it("getActive loads the active organization from /active", async () => {
    const { auth, calls } = setup(() => ({ body: organizationBody }));

    const active = await auth.organization.getActive();

    expect(calls.map((call) => call.url)).toEqual(["http://localhost:8080/auth/organizations/active"]);
    expect(active).toMatchObject({ id: "org-1", name: "Acme" });
    expect(active).not.toHaveProperty("created_at");
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

  it("sets the active organization from an update of the current org", async () => {
    const updated = { ...organizationBody, name: "Acme Updated" };
    const { auth } = setup((req) =>
      req.method === "PATCH" ? { body: updated } : { body: organizationBody },
    );

    await auth.organization.switch({ id: "org-1" });
    await settle();

    await auth.organization.update({ id: "org-1", name: "Acme Updated" });

    expect(auth.organization.active()?.name).toBe("Acme Updated");
  });

  it("does not set active from an update of another org", async () => {
    const { auth } = setup((req) =>
      req.method === "PATCH"
        ? { body: { ...organizationBody, id: "org-2", name: "Other" } }
        : { body: organizationBody },
    );

    await auth.organization.switch({ id: "org-1" });
    await settle();

    await auth.organization.update({ id: "org-2", name: "Other" });

    expect(auth.organization.active()?.name).toBe("Acme");
  });

  it("reloads membership when assigning a role to myself", async () => {
    const { auth, calls } = setup((req) =>
      req.url.includes("/roles/assign") ? { status: 204 } : { body: organizationBody },
    );

    await auth.organization.switch({ id: "org-1" });
    await settle();
    const before = calls.length;

    await auth.organization.assignMemberRole({ memberId: "mem-1", role: "owner" });
    await settle();

    expect(calls.slice(before).map((call) => call.url)).toEqual([
      "http://localhost:8080/auth/organizations/members/mem-1/roles/assign",
      "http://localhost:8080/auth/organizations/me",
    ]);
  });

  it("does not reload membership when assigning a role to someone else", async () => {
    const { auth, calls } = setup((req) =>
      req.url.includes("/roles/assign") ? { status: 204 } : { body: organizationBody },
    );

    await auth.organization.switch({ id: "org-1" });
    await settle();
    const before = calls.length;

    await auth.organization.assignMemberRole({ memberId: "mem-other", role: "member" });
    await settle();

    expect(calls.slice(before).map((call) => call.url)).toEqual([
      "http://localhost:8080/auth/organizations/members/mem-other/roles/assign",
    ]);
  });

  it("clears active stores when removing myself", async () => {
    const { auth } = setup((req) => (req.method === "DELETE" ? { status: 204 } : { body: organizationBody }));

    await auth.organization.switch({ id: "org-1" });
    await settle();
    await auth.organization.getActiveMembership();

    await auth.organization.removeMember({ memberId: "mem-1" });

    expect(auth.organization.active()).toBeNull();
    expect(auth.$activeMembership.get().data).toBeNull();
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
