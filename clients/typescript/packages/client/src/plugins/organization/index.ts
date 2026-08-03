import type { WritableAtom } from "nanostores";
import { atom } from "nanostores";

import { defineClientPlugin, defineRoutes } from "../../define-plugin";
import { camelizeKeys, camelizePage } from "../../helpers";
import { route } from "../../route";
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
  OrganizationPluginConfig,
  OrganizationRole,
  RemoveMemberInput,
  SwitchOrganizationInput,
  UpdateOrganizationInput,
  UpdateOrganizationRoleInput,
} from "./types";

type ActiveOrganizationStore = WritableAtom<Organization | null>;

function clearActiveOrganizationIf(store: ActiveOrganizationStore, affected: string): void {
  const active = store.get();
  if (active !== null && String(active.id) === String(affected)) {
    store.set(null);
  }
}

function parseMember(raw: unknown): Member {
  const member = camelizeKeys<Member>(raw);
  if (member.organization) {
    member.organization = camelizeKeys<Organization>(member.organization);
  }
  if (member.user) {
    member.user = camelizeKeys<EmbeddedUser>(member.user);
  }
  return member;
}

function parseInvitation(raw: unknown): Invitation {
  const invitation = camelizeKeys<Invitation>(raw);
  if (invitation.organization) {
    invitation.organization = camelizeKeys<EmbeddedOrganization>(invitation.organization);
  }
  if (invitation.inviter) {
    invitation.inviter = camelizeKeys<EmbeddedUser>(invitation.inviter);
  }
  return invitation;
}

function buildCoreRoutes($activeOrganization: ActiveOrganizationStore) {
  return defineRoutes(
    route<CreateOrganizationInput, Organization>()({
      method: "POST",
      path: "/",
      as: "organization.create",
      handler: async (_ctx, _input, http) => {
        const organization = await http<Organization>();
        $activeOrganization.set(organization);
        return organization;
      },
    }),
    route<ListOrganizationsInput, Page<Organization>>()({
      method: "GET",
      path: "/",
      as: "organization.list",
    }),
    route<CheckSlugInput, CheckSlugResult>()({
      method: "POST",
      path: "/check-slug",
      as: "organization.checkSlug",
    }),
    route<UpdateOrganizationInput, Organization>()({
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
      handler: async (_ctx, input, http) => {
        await http();
        clearActiveOrganizationIf($activeOrganization, input.id);
      },
    }),
    route<SwitchOrganizationInput, Organization | null>()({
      method: "POST",
      path: "/switch",
      as: "organization.switch",
      serialize: (input) => ({ organization: input.id }),
      handler: async (_ctx, _input, http) => {
        const organization = await http<Organization | null>();
        $activeOrganization.set(organization);
        return organization;
      },
    }),
    route<LeaveOrganizationInput, void>()({
      method: "POST",
      path: "/leave",
      as: "organization.leave",
      serialize: (input) => ({ organization: input.id }),
      handler: async (_ctx, input, http) => {
        await http();
        clearActiveOrganizationIf($activeOrganization, input.id);
      },
    }),
    route<void, Member>()({
      method: "GET",
      path: "/me",
      as: "organization.getActiveMembership",
      parse: parseMember,
      handler: async (_ctx, _input, http) => {
        const member = await http<Member>();
        if (member.organization) {
          $activeOrganization.set(member.organization);
        }
        return member;
      },
    }),
    route<ListMembersInput | void, Page<Member>>()({
      method: "GET",
      path: "/members",
      as: "organization.listMembers",
      parse: (raw) => camelizePage<Member>(raw, parseMember),
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
    route<CreateInvitationInput, Invitation>()({
      method: "POST",
      path: "/invitations",
      as: "organization.invite",
      parse: parseInvitation,
    }),
    route<ListInvitationsInput, Page<Invitation>>()({
      method: "GET",
      path: "/invitations",
      as: "organization.listInvitations",
      parse: (raw) => camelizePage<Invitation>(raw, parseInvitation),
    }),
    route<InvitationTokenInput, Invitation>()({
      method: "GET",
      path: "/invitations/token/:token",
      as: "organization.getInvitation",
      params: ["token"],
      parse: parseInvitation,
    }),
    route<InvitationTokenInput, Invitation>()({
      method: "POST",
      path: "/invitations/respond",
      as: "organization.acceptInvitation",
      serialize: (input) => ({ token: input.token, response: "accept" }),
      parse: parseInvitation,
    }),
    route<InvitationTokenInput, Invitation>()({
      method: "POST",
      path: "/invitations/respond",
      as: "organization.rejectInvitation",
      serialize: (input) => ({ token: input.token, response: "reject" }),
      parse: parseInvitation,
    }),
    route<CancelInvitationInput, Invitation>()({
      method: "POST",
      path: "/invitations/cancel",
      as: "organization.cancelInvitation",
      serialize: (input) => ({ invitation: input.invitationId }),
      parse: parseInvitation,
    }),
  );
}

function buildRoleRoutes() {
  return defineRoutes(
    route<CreateOrganizationRoleInput, OrganizationRole>()({
      method: "POST",
      path: "/roles",
      as: "organization.createRole",
    }),
    route<ListOrganizationRolesInput, Page<OrganizationRole>>()({
      method: "GET",
      path: "/roles",
      as: "organization.listRoles",
    }),
    route<UpdateOrganizationRoleInput, OrganizationRole>()({
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

type CoreRoutes = ReturnType<typeof buildCoreRoutes>;
type RoleRoutes = ReturnType<typeof buildRoleRoutes>;

type OrganizationRoutes<Config extends OrganizationPluginConfig> = Config["customRoles"] extends true
  ? readonly [...CoreRoutes, ...RoleRoutes]
  : CoreRoutes;

export function organizationPlugin<const Config extends OrganizationPluginConfig = Record<never, never>>(
  config?: Config,
) {
  const $activeOrganization = atom<Organization | null>(null);

  const routes = (
    config?.customRoles === true
      ? [...buildCoreRoutes($activeOrganization), ...buildRoleRoutes()]
      : buildCoreRoutes($activeOrganization)
  ) as OrganizationRoutes<Config>;

  return defineClientPlugin({
    id: "organization",
    basePath: "/organizations",
    routes,
    hooks: {
      afterResponse: [
        {
          // The active organization belongs to the session that set it.
          match: ["/signout", "/revoke-sessions"],
          run: (res) => {
            if (res.ok) {
              $activeOrganization.set(null);
            }
            return res;
          },
        },
      ],
    },
    actions: () => ({
      organization: {
        active: (): Organization | null => $activeOrganization.get(),
        activeId: (): string | null => $activeOrganization.get()?.id ?? null,
        $activeOrganization,
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
  OrganizationPermissions,
  OrganizationPluginConfig,
  OrganizationRole,
  RemoveMemberInput,
  SwitchOrganizationInput,
  UpdateOrganizationInput,
  UpdateOrganizationRoleInput,
} from "./types";
