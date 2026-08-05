import type { AnyRouteContext } from "../../context";
import type { DataStore } from "../../data-store";
import type { LimenError } from "../../errors";
import { currentData, storeRef } from "../../stores";
import type { Member, Organization } from "./types";

export const ACTIVE_MEMBERSHIP = "activeMembership";
export const ACTIVE_ORGANIZATION = "activeOrganization";

export const activeMembershipStore = storeRef<Member | null>(ACTIVE_MEMBERSHIP);

export const activeOrganizationStore = storeRef<Organization | null>(ACTIVE_ORGANIZATION);

export function activeOrganization(ctx: AnyRouteContext) {
  const store: DataStore<Organization | null> | undefined = ctx.stores.get(activeOrganizationStore);
  const current = (): Organization | null => currentData(ctx, activeOrganizationStore) ?? null;

  return {
    current,
    set: (organization: Organization | null): void => store?.setData(organization),
    reload: async (): Promise<void> => {
      await store?.refetch({ force: true });
    },
    error: (): LimenError | null => store?.current().error ?? null,
  };
}

export function clearActiveIf(ctx: AnyRouteContext, organizationId: string): void {
  const active = activeOrganization(ctx);
  const current = active.current();
  if (current !== null && current.id === organizationId) {
    active.set(null);
    membership(ctx).clear();
  }
}

export function isActiveOrganization(ctx: AnyRouteContext, organizationId: string): boolean {
  return activeOrganization(ctx).current()?.id === organizationId;
}

export function isActiveMembership(ctx: AnyRouteContext, memberId: string): boolean {
  return membership(ctx).current()?.id === memberId;
}

export function holdsRole(ctx: AnyRouteContext, roleName: string): boolean {
  return membership(ctx).current()?.roles?.includes(roleName) === true;
}

export function membership(ctx: AnyRouteContext) {
  const store: DataStore<Member | null> | undefined = ctx.stores.get(activeMembershipStore);
  const current = (): Member | null => currentData(ctx, activeMembershipStore) ?? null;

  return {
    current,
    set: (member: Member | null): void => store?.setData(member),
    reload: async (): Promise<void> => {
      await store?.refetch({ force: true });
    },
    clear: (): void => store?.setData(null),
    error: (): LimenError | null => store?.current().error ?? null,
  };
}
