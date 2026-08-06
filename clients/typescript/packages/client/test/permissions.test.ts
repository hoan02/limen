import { describe, expect, it } from "vitest";
import { can, canAny } from "../src/permissions";

const granted = ["invitation:create", "member:read", "member:update"];

describe("can", () => {
  it("requires every action in a comma-separated spec", () => {
    expect(can(granted, "member:read,update")).toBe(true);
    expect(can(granted, "member:read,remove")).toBe(false);
  });

  it("requires every spec in an array", () => {
    expect(can(granted, ["invitation:create", "member:read"])).toBe(true);
    expect(can(granted, ["invitation:create", "member:remove"])).toBe(false);
  });

  it("treats a granted wildcard as every action on that resource", () => {
    expect(can(["member:*"], "member:remove")).toBe(true);
    expect(can(["member:*"], "invitation:create")).toBe(false);
  });

  it("satisfies a required wildcard only with a granted wildcard", () => {
    expect(can(["member:*"], "member:*")).toBe(true);
    expect(can(["member:read"], "member:*")).toBe(false);
  });

  it("denies when nothing is required", () => {
    expect(can(granted, [])).toBe(false);
  });

  it("reads permissions off a source that carries them", () => {
    expect(can({ permissions: granted }, "member:read")).toBe(true);
    expect(can({ permissions: undefined }, "member:read")).toBe(false);
    expect(can({}, "member:read")).toBe(false);
    expect(can(null, "member:read")).toBe(false);
    expect(can(undefined, "member:read")).toBe(false);
  });

  it("ignores surrounding whitespace in a spec", () => {
    expect(can(granted, " member : read , update ")).toBe(true);
  });

  it("rejects a malformed spec rather than denying silently", () => {
    expect(() => can(granted, "memberread")).toThrow(/invalid permission spec/);
    expect(() => can(granted, "member:")).toThrow(/invalid permission spec/);
    expect(() => can(granted, ":read")).toThrow(/invalid permission spec/);
    expect(() => can(granted, "member:read,")).toThrow(/empty action/);
  });
});

describe("canAny", () => {
  it("passes when one required action is granted", () => {
    expect(canAny(granted, ["member:remove", "member:read"])).toBe(true);
    expect(canAny(granted, "member:remove,read")).toBe(true);
    expect(canAny(granted, ["member:remove", "organization:delete"])).toBe(false);
  });

  it("denies when nothing is required", () => {
    expect(canAny(granted, [])).toBe(false);
  });
});
