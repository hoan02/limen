import { createAuthStore } from "../../auth-store";
import type { DeclaredFields, StrictConfig } from "../../define-plugin";
import { defineClientPlugin, defineFields, defineStores } from "../../define-plugin";
import { camelizeKeys } from "../../helpers";
import { sessionStore } from "../../session-store";
import { effect } from "../../stores";
import { parseMember } from "./parse";
import { buildCoreRoutes, buildRoleRoutes, type OrganizationRoutes } from "./routes";
import { ACTIVE_MEMBERSHIP, ACTIVE_ORGANIZATION, activeOrganization, membership } from "./stores";
import type { Member, Organization, OrganizationModelFields, OrganizationPluginConfig } from "./types";

/**
 * Declare the app's own columns on the organization models — type-only. Pass
 * the result as the plugin's `fields`.
 *
 * @example
 *   organizationPlugin({ fields: fields<{ organization: { industry: string } }>() })
 */
export const fields = defineFields<OrganizationModelFields>();

export function organizationPlugin<
  const Config extends StrictConfig<Config, OrganizationPluginConfig> = Record<never, never>,
>(config?: Config) {
  type F = DeclaredFields<Config>;

  const routes = (
    config?.customRoles === true ? [...buildCoreRoutes<F>(), ...buildRoleRoutes<F>()] : buildCoreRoutes<F>()
  ) as OrganizationRoutes<Config, F>;

  return defineClientPlugin({
    id: "organization",
    basePath: "/organizations",
    routes,
    stores: (ctx) =>
      defineStores({
        [ACTIVE_ORGANIZATION]: createAuthStore({
          ctx,
          path: "/active",
          parse: (raw) => camelizeKeys<Organization<F>>(raw),
        }),
        [ACTIVE_MEMBERSHIP]: createAuthStore({
          ctx,
          path: "/me",
          parse: parseMember<F>,
        }),
      }),
    effects: [
      effect(sessionStore, (session, ctx) => {
        if (session.data === null) {
          activeOrganization(ctx).set(null);
          membership(ctx).clear();
        }
      }),
    ],
    actions: (ctx) => ({
      organization: {
        active: (): Organization<F> | null => activeOrganization(ctx).current() as Organization<F> | null,
        activeId: (): string | null => activeOrganization(ctx).current()?.id ?? null,
        /** Reload the active organization, and return it — `null` when there is none. */
        getActive: async (): Promise<Organization<F> | null> => {
          const store = activeOrganization(ctx);
          await store.reload();
          const error = store.error();
          if (error !== null) {
            throw error;
          }
          return store.current() as Organization<F> | null;
        },
        /** Reload and return the active membership, or `null` when there is none. */
        getActiveMembership: async (): Promise<Member<F> | null> => {
          const store = membership(ctx);
          await store.reload();
          const error = store.error();
          if (error !== null) {
            throw error;
          }
          return store.current() as Member<F> | null;
        },
      },
    }),
  });
}

export { activeMembershipStore, activeOrganizationStore } from "./stores";
export type {
  CancelInvitationInput,
  CheckSlugInput,
  CheckSlugResult,
  CreateInvitationInput,
  CreateOrganizationInput,
  CreateOrganizationRoleInput,
  DeleteOrganizationInput,
  DeleteOrganizationRoleInput,
  EmbeddedOrganization,
  EmbeddedUser,
  Invitation,
  InvitationStatus,
  InvitationTokenInput,
  LeaveOrganizationInput,
  ListInvitationsInput,
  ListMembersInput,
  ListOrganizationRolesInput,
  ListOrganizationsInput,
  Member,
  MemberRoleInput,
  Organization,
  OrganizationModelFields,
  OrganizationPermissions,
  OrganizationPluginConfig,
  OrganizationRole,
  RemoveMemberInput,
  SwitchOrganizationInput,
  UpdateOrganizationInput,
  UpdateOrganizationRoleInput,
} from "./types";
