import { defineClientPlugin, defineRoutes } from "../../define-plugin";
import { camelizeKeys, camelizePage } from "../../helpers";
import { route } from "../../route";
import type { Page } from "../../types";
import type {
  ApiKey,
  CreateApiKeyInput,
  CreateApiKeyResult,
  GetApiKeyInput,
  ListApiKeysInput,
  RevokeApiKeyInput,
  RotateApiKeyInput,
  UpdateApiKeyInput,
} from "./types";

function parseApiKey<T extends ApiKey>(raw: unknown): T {
  return camelizeKeys<T>(raw);
}

export function apiKeyPlugin() {
  const routes = defineRoutes(
    route<CreateApiKeyInput, CreateApiKeyResult>()({
      method: "POST",
      path: "/",
      as: "apiKey.create",
      parse: (raw) => parseApiKey<CreateApiKeyResult>(raw),
    }),
    route<ListApiKeysInput, Page<ApiKey>>()({
      method: "GET",
      path: "/",
      as: "apiKey.list",
      parse: (raw) => camelizePage<ApiKey>(raw),
    }),
    route<GetApiKeyInput, ApiKey>()({
      method: "GET",
      path: "/:id",
      as: "apiKey.get",
      params: ["id"],
      parse: parseApiKey,
    }),
    route<UpdateApiKeyInput, ApiKey>()({
      method: "PATCH",
      path: "/:id",
      as: "apiKey.update",
      params: ["id"],
      parse: parseApiKey,
    }),
    route<RevokeApiKeyInput, void>()({
      method: "DELETE",
      path: "/:id",
      as: "apiKey.revoke",
      params: ["id"],
    }),
    route<RotateApiKeyInput, CreateApiKeyResult>()({
      method: "POST",
      path: "/:id/rotate",
      as: "apiKey.rotate",
      params: ["id"],
      parse: (raw) => parseApiKey<CreateApiKeyResult>(raw),
    }),
  );

  return defineClientPlugin({
    id: "api-key",
    basePath: "/api-keys",
    routes,
  });
}

export type {
  ApiKey,
  ApiKeyPermissions,
  CreateApiKeyInput,
  CreateApiKeyResult,
  GetApiKeyInput,
  ListApiKeysInput,
  RevokeApiKeyInput,
  RotateApiKeyInput,
  UpdateApiKeyInput,
} from "./types";
