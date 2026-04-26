import { CombinedError, createClient } from "urql";
import { filter, map, pipe } from "wonka";
import type { Exchange, Operation } from "@urql/core";

export type MockHandler = (
  op: Operation,
) => { data?: unknown; error?: CombinedError };

function mockExchange(handler: MockHandler): Exchange {
  return () => (ops$) =>
    pipe(
      ops$,
      filter((op) => op.kind !== "teardown"),
      map((op) => ({
        operation: op,
        hasNext: false,
        stale: false,
        ...handler(op),
      })),
    );
}

export function createTestClient(handler: MockHandler) {
  return createClient({
    url: "http://localhost/graphql",
    exchanges: [mockExchange(handler)],
  });
}
