import { defineRoutes } from "../../define-plugin";
import { camelizePage } from "../../helpers";
import { route } from "../../route";
import type { Page } from "../../types";
import { parseInvitation, parseMember } from "./parse";
import {
  activeOrganization,
  clearActiveIf,
  holdsRole,
  isActiveOrganization,
  isActiveMembership,
  membership,
} from "./stores";
import type {
  CancelInvitationInput,
  CheckSlugInput,
  CheckSlugResult,
  CreateInvitationInput,
  CreateOrganizationInput,
  CreateOrganizationRoleInput,
  DeleteOrganizationInput,
  DeleteOrganizationRoleInput,
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
  OrganizationPluginConfig,
  OrganizationRole,
  RemoveMemberInput,
  SwitchOrganizationInput,
  UpdateOrganizationInput,
  UpdateOrganizationRoleInput,
} from "./types";

export function buildCoreRoutes<F>() {
  return defineRoutes(
    route<CreateOrganizationInput, Organization<F>>()({
      method: "POST",
      path: "/",
      as: "organization.create",
      handler: async (ctx, _input, http) => {
        const organization = await http<Organization<F>>();
        activeOrganization(ctx).set(organization);
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
      handler: async (ctx, input, http) => {
        const organization = await http<Organization<F>>();
        if (isActiveOrganization(ctx, input.id)) {
          activeOrganization(ctx).set(organization);
        }
        return organization;
      },
    }),
    route<DeleteOrganizationInput, void>()({
      method: "DELETE",
      path: "/:id",
      as: "organization.delete",
      params: ["id"],
      handler: async (ctx, input, http) => {
        await http();
        clearActiveIf(ctx, input.id);
      },
    }),
    route<SwitchOrganizationInput, Organization<F> | null>()({
      method: "POST",
      path: "/switch",
      as: "organization.switch",
      serialize: (input) => ({ organization: input.id }),
      handler: async (ctx, _input, http) => {
        const organization = await http<Organization<F> | null>();
        activeOrganization(ctx).set(organization);
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
        clearActiveIf(ctx, input.id);
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
      handler: async (ctx, input, http) => {
        await http();
        if (isActiveMembership(ctx, input.memberId)) {
          void membership(ctx).reload();
        }
      },
    }),
    route<MemberRoleInput, void>()({
      method: "POST",
      path: "/members/:memberId/roles/revoke",
      as: "organization.revokeMemberRole",
      params: ["memberId"],
      serialize: (input) => ({ roles: [input.role] }),
      handler: async (ctx, input, http) => {
        await http();
        if (isActiveMembership(ctx, input.memberId)) {
          void membership(ctx).reload();
        }
      },
    }),
    route<RemoveMemberInput, void>()({
      method: "DELETE",
      path: "/members/:memberId",
      as: "organization.removeMember",
      params: ["memberId"],
      handler: async (ctx, input, http) => {
        await http();
        if (isActiveMembership(ctx, input.memberId)) {
          activeOrganization(ctx).set(null);
          membership(ctx).clear();
        }
      },
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

export function buildRoleRoutes<F>() {
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
      handler: async (ctx, _input, http) => {
        const role = await http<OrganizationRole<F>>();
        if (holdsRole(ctx, role.name)) {
          void membership(ctx).reload();
        }
        return role;
      },
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

export type OrganizationRoutes<Config extends OrganizationPluginConfig, F> = Config["customRoles"] extends true
  ? readonly [...CoreRoutes<F>, ...RoleRoutes<F>]
  : CoreRoutes<F>;
