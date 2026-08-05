import { expectTypeOf, test } from "vitest";
import { createAuthClient } from "../src";
import { fields, organizationPlugin } from "../src/plugins/organization";

test("fields extends organization and member responses", () => {
  const auth = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins: [
      organizationPlugin({
        fields: fields<{ organization: { industry: string }; member: { title: string | null } }>(),
      }),
    ],
  });

  expectTypeOf(auth.organization.create).returns.resolves.toHaveProperty("industry");
  expectTypeOf(auth.organization.getActiveMembership).returns.resolves.toExtend<{
    title: string | null;
  } | null>();
  expectTypeOf(auth.organization.active()).toExtend<{ industry: string } | null>();
});

test("rejects undeclared plugin config keys and unknown models", () => {
  organizationPlugin({
    customRoles: true,
    fields: fields<{ organization: { industry: string } }>(),
    // @ts-expect-error `bogus` is not a config option
    bogus: 1,
  });

  // @ts-expect-error `organisation` is not one of the organization models
  organizationPlugin({ fields: fields<{ organisation: { industry: string } }>() });
});
