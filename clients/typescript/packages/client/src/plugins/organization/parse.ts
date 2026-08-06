import { camelizeKeys } from "../../helpers";
import type { EmbeddedOrganization, EmbeddedUser, Invitation, Member, Organization } from "./types";

export function parseMember<F>(raw: unknown): Member<F> {
  const member = camelizeKeys<Member<F>>(raw);
  if (member.organization) {
    member.organization = camelizeKeys<Organization<F>>(member.organization);
  }
  if (member.user) {
    member.user = camelizeKeys<EmbeddedUser>(member.user);
  }
  return member;
}

export function parseInvitation<F>(raw: unknown): Invitation<F> {
  const invitation = camelizeKeys<Invitation<F>>(raw);
  if (invitation.organization) {
    invitation.organization = camelizeKeys<EmbeddedOrganization<F>>(invitation.organization);
  }
  if (invitation.inviter) {
    invitation.inviter = camelizeKeys<EmbeddedUser>(invitation.inviter);
  }
  return invitation;
}
