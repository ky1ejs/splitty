import { Client, cacheExchange, fetchExchange } from "urql";
import { authExchange } from "@urql/exchange-auth";
import {
  getAccessToken,
  getRefreshToken,
  setTokens,
  clearTokens,
} from "../auth/token-store";
import { RefreshTokenMutation } from "./operations";

const GRAPHQL_URL =
  import.meta.env.VITE_GRAPHQL_URL ?? "http://localhost:8080/query";

export const client = new Client({
  url: GRAPHQL_URL,
  exchanges: [
    cacheExchange,
    authExchange(async (utils) => {
      return {
        addAuthToOperation(operation) {
          const token = getAccessToken();
          if (!token) return operation;
          return utils.appendHeaders(operation, {
            Authorization: `Bearer ${token}`,
          });
        },
        willAuthError() {
          return !getAccessToken();
        },
        didAuthError(error) {
          return (
            error.graphQLErrors.some(
              (e) => e.extensions?.["code"] === "UNAUTHORIZED",
            ) || error.response?.status === 401
          );
        },
        async refreshAuth() {
          const refreshToken = getRefreshToken();
          if (!refreshToken) {
            clearTokens();
            return;
          }

          const result = await utils.mutate(RefreshTokenMutation, {
            refreshToken,
          });

          if (result.data?.refreshToken) {
            setTokens(
              result.data.refreshToken.accessToken,
              result.data.refreshToken.refreshToken,
            );
          } else {
            clearTokens();
          }
        },
      };
    }),
    fetchExchange,
  ],
});
