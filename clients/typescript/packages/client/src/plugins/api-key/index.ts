import { defineClientPlugin, defineRoutes } from "../../define-plugin";
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

export function apiKeyPlugin() {
  const routes = defineRoutes(
    route<CreateApiKeyInput, CreateApiKeyResult>()({
      method: "POST",
      path: "/",
      as: "apiKey.create",
    }),
    route<ListApiKeysInput, Page<ApiKey>>()({
      method: "GET",
      path: "/",
      as: "apiKey.list",
    }),
    route<GetApiKeyInput, ApiKey>()({
      method: "GET",
      path: "/:id",
      as: "apiKey.get",
      params: ["id"],
    }),
    route<UpdateApiKeyInput, ApiKey>()({
      method: "PATCH",
      path: "/:id",
      as: "apiKey.update",
      params: ["id"],
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
