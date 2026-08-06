const WILDCARD = "*";

/** `"resource:action"`, actions optionally comma-separated: `"invitation:create,read"`. */
export type PermissionInput = string | readonly string[];

export type PermissionSource = readonly string[] | { permissions?: readonly string[] | undefined } | null | undefined;

type Requirement = {
  resource: string;
  action: string;
};

function isPermissionList(source: NonNullable<PermissionSource>): source is readonly string[] {
  return Array.isArray(source);
}

function grantedPermissions(source: PermissionSource): readonly string[] {
  if (source === null || source === undefined) {
    return [];
  }
  return isPermissionList(source) ? source : (source.permissions ?? []);
}

function parseSpec(spec: string): Requirement[] {
  const trimmed = spec.trim();
  const separator = trimmed.indexOf(":");
  if (separator === -1) {
    throw new Error(`limen: invalid permission spec "${spec}"`);
  }

  const resource = trimmed.slice(0, separator).trim();
  const actions = trimmed.slice(separator + 1).trim();
  if (resource === "" || actions === "") {
    throw new Error(`limen: invalid permission spec "${spec}"`);
  }

  return actions.split(",").map((action) => {
    const name = action.trim();
    if (name === "") {
      throw new Error(`limen: invalid permission spec "${spec}": empty action`);
    }
    return { resource, action: name };
  });
}

function requirements(required: PermissionInput): Requirement[] {
  const specs = typeof required === "string" ? [required] : required;
  return specs.flatMap(parseSpec);
}

function permits(granted: readonly string[], requirement: Requirement): boolean {
  const { resource, action } = requirement;
  return granted.includes(`${resource}:${action}`) || granted.includes(`${resource}:${WILDCARD}`);
}

/**
 * Whether `source` grants every action in `required`.
 *
 * Requiring nothing denies. Throws when a spec is not `"resource:action"`.
 */
export function can(source: PermissionSource, required: PermissionInput): boolean {
  const needed = requirements(required);
  if (needed.length === 0) {
    return false;
  }

  const granted = grantedPermissions(source);
  return needed.every((requirement) => permits(granted, requirement));
}

/** Whether `source` grants at least one action in `required`. Same contract as {@link can}. */
export function canAny(source: PermissionSource, required: PermissionInput): boolean {
  const needed = requirements(required);
  if (needed.length === 0) {
    return false;
  }

  const granted = grantedPermissions(source);
  return needed.some((requirement) => permits(granted, requirement));
}
