import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { CombinedError, createClient, Provider } from "urql";
import { map, pipe } from "wonka";
import { describe, expect, it, vi } from "vitest";
import { GroupDetailPage } from "./group-detail";
import type { Exchange } from "@urql/core";

vi.mock("../auth/auth-context", () => ({
  useAuth: () => ({
    user: { id: "1", email: "test@test.com", displayName: "Test User" },
    logout: vi.fn(),
    isAuthenticated: true,
    loading: false,
    login: vi.fn(),
  }),
}));

function mockExchange(response: { data?: unknown; error?: CombinedError }): Exchange {
  return () => (ops$) =>
    pipe(
      ops$,
      map((op) => ({
        operation: op,
        data: response.data,
        error: response.error,
        hasNext: false,
        stale: false,
      })),
    );
}

function createTestClient(response: { data?: unknown; error?: CombinedError }) {
  return createClient({
    url: "http://localhost/graphql",
    exchanges: [mockExchange(response)],
  });
}

function renderAtRoute(
  client: ReturnType<typeof createTestClient>,
  path = "/groups/g1",
) {
  return render(
    <Provider value={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/groups/:id" element={<GroupDetailPage />} />
        </Routes>
      </MemoryRouter>
    </Provider>,
  );
}

describe("GroupDetailPage", () => {
  it("renders group details", () => {
    const client = createTestClient({
      data: {
        group: {
          __typename: "Group",
          id: "g1",
          name: "Trip to Paris",
          createdAt: "2026-01-15T00:00:00Z",
          createdBy: {
            __typename: "User",
            id: "1",
            displayName: "Test User",
          },
          members: [
            {
              __typename: "User",
              id: "1",
              email: "test@test.com",
              displayName: "Test User",
            },
            {
              __typename: "User",
              id: "2",
              email: "other@test.com",
              displayName: "Other User",
            },
          ],
        },
      },
    });
    renderAtRoute(client);

    expect(screen.getByText("Trip to Paris")).toBeInTheDocument();
    expect(screen.getByText(/Created by Test User/)).toBeInTheDocument();
    expect(
      screen.getByText("Test User (test@test.com)"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Other User (other@test.com)"),
    ).toBeInTheDocument();
  });

  it("shows not found when group is null", () => {
    const client = createTestClient({ data: { group: null } });
    renderAtRoute(client);

    expect(screen.getByText("Group not found.")).toBeInTheDocument();
  });

  it("shows error message on query failure", () => {
    const client = createTestClient({
      error: new CombinedError({
        networkError: new Error("Not authorized"),
      }),
    });
    renderAtRoute(client);

    expect(
      screen.getByText(/Failed to load group/),
    ).toBeInTheDocument();
  });

  it("has a back link to the groups list", () => {
    const client = createTestClient({
      data: {
        group: {
          __typename: "Group",
          id: "g1",
          name: "Trip to Paris",
          createdAt: "2026-01-15T00:00:00Z",
          createdBy: {
            __typename: "User",
            id: "1",
            displayName: "Test User",
          },
          members: [],
        },
      },
    });
    renderAtRoute(client);

    const backLink = screen.getByText("\u2190 Back to groups");
    expect(backLink.closest("a")).toHaveAttribute("href", "/");
  });
});
