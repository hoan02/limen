import { describe, expect, it } from "vitest";
import { defaultSerialize } from "../src/serialize";

describe("defaultSerialize", () => {
  it("snake_cases known fields and flattens additionalFields", () => {
    expect(
      defaultSerialize({
        name: "Acme",
        expiresIn: 60,
        additionalFields: { industry: "fintech" },
      }),
    ).toEqual({ name: "Acme", expires_in: 60, industry: "fintech" });
  });
});
