import type { AnyRouteContext } from "../../context";
import type { DataStore } from "../../data-store";
import { createStore } from "../../data-store";
import type { DeclaredFields, StrictConfig } from "../../define-plugin";
import { defineClientPlugin, defineFields, defineRoutes, defineStores } from "../../define-plugin";
import { LimenError } from "../../errors";
import { camelizeKeys, camelizePage } from "../../helpers";
import { route } from "../../route";
import { sessionStore } from "../../session-store";
import { currentData, effect, storeRef } from "../../stores";
import type { Page } from "../../types";
import type {
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
  OrganizationPluginConfig,
  OrganizationRole,
  RemoveMemberInput,
  SwitchOrganizationInput,
  UpdateOrganizationInput,
  UpdateOrganizationRoleInput,
} from "./types";

const ACTIVE_MEMBERSHIP = "activeMembership";

/**
 * Declare the app's own columns on the organization models — type-only. Pass
 * the result as the plugin's `fields`.
 *
 * @example
 *   organizationPlugin({ fields: fields<{ organization: { industry: string } }>() })
 */
export const fields = defineFields<OrganizationModelFields>();

/**
 * The session's membership in its active organization, published for other
 * plugins. Typed with the base models: app `fields` do not reach ref consumers.
 */
export const activeMembershipStore = storeRef<Member | null>(ACTIVE_MEMBERSHIP);

function membership(ctx: AnyRouteContext) {
  const store: DataStore<Member | null> | undefined = ctx.stores.get(activeMembershipStore);
  const current = (): Member | null => currentData(ctx, activeMembershipStore) ?? null;

  return {
    current,
    organization: (): Organization | null => current()?.organization ?? null,
    set: (member: Member | null): void => store?.setData(member),
    reload: async (): Promise<void> => {
      await store?.refetch({ force: true });
    },
    clearIf: (organizationId: string): void => {
      const active = current()?.organization;
      if (active !== undefined && String(active.id) === String(organizationId)) {
        store?.setData(null);
      }
    },
    error: (): LimenError | null => store?.current().error ?? null,
  };
}

function parseMember<F>(raw: unknown): Member<F> {
  const member = camelizeKeys<Member<F>>(raw);
  if (member.organization) {
    member.organization = camelizeKeys<Organization<F>>(member.organization);
  }
  if (member.user) {
    member.user = camelizeKeys<EmbeddedUser>(member.user);
  }
  return member;
}

function parseInvitation<F>(raw: unknown): Invitation<F> {
  const invitation = camelizeKeys<Invitation<F>>(raw);
  if (invitation.organization) {
    invitation.organization = camelizeKeys<EmbeddedOrganization<F>>(invitation.organization);
  }
  if (invitation.inviter) {
    invitation.inviter = camelizeKeys<EmbeddedUser>(invitation.inviter);
  }
  return invitation;
}

function buildCoreRoutes<F>() {
  return defineRoutes(
    route<CreateOrganizationInput, Organization<F>>()({
      method: "POST",
      path: "/",
      as: "organization.create",
      handler: async (ctx, _input, http) => {
        const organization = await http<Organization<F>>();
        void membership(ctx).reload();
        return organization;
      },
    }),
    route<ListOrganizationsInput, Page<Organization<F>>>()({
      method: "GET",
      path: "/",
      as: "organization.list",
    }),
    route<CheckSlugInput, CheckSlugResult>()({
      method: "POST",
      path: "/check-slug",
      as: "organization.checkSlug",
    }),
    route<UpdateOrganizationInput, Organization<F>>()({
      method: "PATCH",
      path: "/:id",
      as: "organization.update",
      params: ["id"],
    }),
    route<DeleteOrganizationInput, void>()({
      method: "DELETE",
      path: "/:id",
      as: "organization.delete",
      params: ["id"],
      handler: async (ctx, input, http) => {
        await http();
        membership(ctx).clearIf(input.id);
      },
    }),
    route<SwitchOrganizationInput, Organization<F> | null>()({
      method: "POST",
      path: "/switch",
      as: "organization.switch",
      serialize: (input) => ({ organization: input.id }),
      handler: async (ctx, _input, http) => {
        const organization = await http<Organization<F> | null>();
        if (organization === null) {
          membership(ctx).set(null);
        } else {
          void membership(ctx).reload();
        }
        return organization;
      },
    }),
    route<LeaveOrganizationInput, void>()({
      method: "POST",
      path: "/leave",
      as: "organization.leave",
      serialize: (input) => ({ organization: input.id }),
      handler: async (ctx, input, http) => {
        await http();
        membership(ctx).clearIf(input.id);
      },
    }),
    route<ListMembersInput | void, Page<Member<F>>>()({
      method: "GET",
      path: "/members",
      as: "organization.listMembers",
      parse: (raw) => camelizePage<Member<F>>(raw, parseMember),
    }),
    route<MemberRoleInput, void>()({
      method: "POST",
      path: "/members/:memberId/roles/assign",
      as: "organization.assignMemberRole",
      params: ["memberId"],
      serialize: (input) => ({ roles: [input.role] }),
    }),
    route<MemberRoleInput, void>()({
      method: "POST",
      path: "/members/:memberId/roles/revoke",
      as: "organization.revokeMemberRole",
      params: ["memberId"],
      serialize: (input) => ({ roles: [input.role] }),
    }),
    route<RemoveMemberInput, void>()({
      method: "DELETE",
      path: "/members/:memberId",
      as: "organization.removeMember",
      params: ["memberId"],
    }),
    route<CreateInvitationInput, Invitation<F>>()({
      method: "POST",
      path: "/invitations",
      as: "organization.invite",
      parse: parseInvitation<F>,
    }),
    route<ListInvitationsInput, Page<Invitation<F>>>()({
      method: "GET",
      path: "/invitations",
      as: "organization.listInvitations",
      parse: (raw) => camelizePage<Invitation<F>>(raw, parseInvitation),
    }),
    route<InvitationTokenInput, Invitation<F>>()({
      method: "GET",
      path: "/invitations/token/:token",
      as: "organization.getInvitation",
      params: ["token"],
      parse: parseInvitation<F>,
    }),
    route<InvitationTokenInput, Invitation<F>>()({
      method: "POST",
      path: "/invitations/respond",
      as: "organization.acceptInvitation",
      serialize: (input) => ({ token: input.token, response: "accept" }),
      parse: parseInvitation<F>,
    }),
    route<InvitationTokenInput, Invitation<F>>()({
      method: "POST",
      path: "/invitations/respond",
      as: "organization.rejectInvitation",
      serialize: (input) => ({ token: input.token, response: "reject" }),
      parse: parseInvitation<F>,
    }),
    route<CancelInvitationInput, Invitation<F>>()({
      method: "POST",
      path: "/invitations/cancel",
      as: "organization.cancelInvitation",
      serialize: (input) => ({ invitation: input.invitationId }),
      parse: parseInvitation<F>,
    }),
  );
}

function buildRoleRoutes<F>() {
  return defineRoutes(
    route<CreateOrganizationRoleInput, OrganizationRole<F>>()({
      method: "POST",
      path: "/roles",
      as: "organization.createRole",
    }),
    route<ListOrganizationRolesInput, Page<OrganizationRole<F>>>()({
      method: "GET",
      path: "/roles",
      as: "organization.listRoles",
    }),
    route<UpdateOrganizationRoleInput, OrganizationRole<F>>()({
      method: "PATCH",
      path: "/roles/:roleId",
      as: "organization.updateRole",
      params: ["roleId"],
    }),
    route<DeleteOrganizationRoleInput, void>()({
      method: "DELETE",
      path: "/roles/:roleId",
      as: "organization.deleteRole",
      params: ["roleId"],
    }),
  );
}

type CoreRoutes<F> = ReturnType<typeof buildCoreRoutes<F>>;
type RoleRoutes<F> = ReturnType<typeof buildRoleRoutes<F>>;

type OrganizationRoutes<Config extends OrganizationPluginConfig, F> = Config["customRoles"] extends true
  ? readonly [...CoreRoutes<F>, ...RoleRoutes<F>]
  : CoreRoutes<F>;

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
        [ACTIVE_MEMBERSHIP]: createStore<Member<F> | null>({
          initial: null,
          fetchOnMount: true,
          loader: async () => {
            // No session means no organization to load.
            const session = ctx.store.current();
            if (session.settled && session.data === null) {
              return null;
            }

            try {
              return parseMember<F>(await ctx.fetch<unknown>("/me", { method: "GET" }));
            } catch (error) {
              // 401 here is "no active organization", not signed out.
              if (error instanceof LimenError && error.isUnauthorized) {
                return null;
              }
              throw error;
            }
          },
        }),
      }),
    effects: [
      effect(sessionStore, (session, ctx) => {
        if (session.data === null) {
          membership(ctx).set(null);
        }
      }),
    ],
    actions: (ctx) => ({
      organization: {
        active: (): Organization<F> | null => membership(ctx).organization() as Organization<F> | null,
        activeId: (): string | null => membership(ctx).organization()?.id ?? null,
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
