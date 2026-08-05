import type { PaginationInput } from "../../types";

export type ApiKeyPermissions = Record<string, string[]>;

export type ApiKey = {
  id: string | number;
  name: string;
  profile: string;
  prefix: string | null;
  last4: string;
  permissions: ApiKeyPermissions;
  enabled: boolean;
  expiresAt: string | null;
  isExpired: boolean;
  lastUsedAt: string | null;
  metadata: Record<string, unknown> | null;
  createdAt: string;
  updatedAt: string;
};

export type CreateApiKeyInput = {
  profile?: string;
  name: string;
  permissions?: ApiKeyPermissions;
  expiresIn?: number;
  metadata?: Record<string, unknown>;
};

export type CreateApiKeyResult = ApiKey & {
  /** Plaintext secret. It is only returned when the key is created or rotated. */
  key: string;
};

export type ListApiKeysInput =
  | (PaginationInput & {
      profile?: string;
      status?: "enabled" | "disabled";
    })
  | void;

export type GetApiKeyInput = {
  id: string | number;
};

export type UpdateApiKeyInput = GetApiKeyInput & {
  name?: string;
  permissions?: ApiKeyPermissions;
  allPermissions?: boolean;
  enabled?: boolean;
  metadata?: Record<string, unknown>;
};

export type RevokeApiKeyInput = GetApiKeyInput;

export type RotateApiKeyInput = GetApiKeyInput & {
  /** New lifetime in seconds, or `null` to remove the expiration. */
  expiresIn: number | null;
  permissions?: ApiKeyPermissions;
  allPermissions?: boolean;
};
