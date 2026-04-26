import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { Provider } from "urql";
import { describe, expect, it, vi } from "vitest";
import { HomePage } from "./home";
import { createTestClient } from "../test-helpers";

const mockLogout = vi.fn();
const mockUser = { id: "1", email: "test@test.com", displayName: "Test User" };

vi.mock("../auth/auth-context", () => ({
  useAuth: () => ({
    user: mockUser,
    logout: mockLogout,
    isAuthenticated: true,
    loading: false,
    login: vi.fn(),
  }),
}));

function renderWithProviders(
  client: ReturnType<typeof createTestClient>,
) {
  return render(
    <Provider value={client}>
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>
    </Provider>,
  );
}

const emptyGroupsHandler = () => ({ data: { groups: [] } });

const groupsHandler = () => ({
  data: {
    groups: [
      {
        __typename: "Group" as const,
        id: "g1",
        name: "Trip to Paris",
        createdAt: "2026-01-01T00:00:00Z",
        createdBy: {
          __typename: "User" as const,
          id: "1",
          displayName: "Test User",
        },
        members: [
          {
            __typename: "User" as const,
            id: "1",
            displayName: "Test User",
          },
          {
            __typename: "User" as const,
            id: "2",
            displayName: "Other User",
          },
        ],
      },
      {
        __typename: "Group" as const,
        id: "g2",
        name: "Shared House",
        createdAt: "2026-02-01T00:00:00Z",
        createdBy: {
          __typename: "User" as const,
          id: "1",
          displayName: "Test User",
        },
        members: [
          {
            __typename: "User" as const,
            id: "1",
            displayName: "Test User",
          },
        ],
      },
    ],
  },
});

describe("HomePage", () => {
  it("shows empty state when there are no groups", () => {
    const client = createTestClient(emptyGroupsHandler);
    renderWithProviders(client);

    expect(
      screen.getByText("No groups yet. Create one below!"),
    ).toBeInTheDocument();
  });

  it("renders a list of groups", () => {
    const client = createTestClient(groupsHandler);
    renderWithProviders(client);

    expect(screen.getByText("Trip to Paris")).toBeInTheDocument();
    expect(screen.getByText("2 members")).toBeInTheDocument();
    expect(screen.getByText("Shared House")).toBeInTheDocument();
    expect(screen.getByText("1 member")).toBeInTheDocument();
  });

  it("links each group to its detail page", () => {
    const client = createTestClient(groupsHandler);
    renderWithProviders(client);

    const link = screen.getByText("Trip to Paris").closest("a");
    expect(link).toHaveAttribute("href", "/groups/g1");
  });

  it("shows the create group form", () => {
    const client = createTestClient(emptyGroupsHandler);
    renderWithProviders(client);

    expect(screen.getByLabelText("Group name")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Create group" }),
    ).toBeInTheDocument();
  });

  it("submits the create group form", async () => {
    const user = userEvent.setup();
    const mutationSpy = vi.fn();
    const client = createTestClient((op) => {
      if (op.kind === "mutation") {
        mutationSpy(op);
        return {
          data: { createGroup: { id: "g-new", name: "New Group" } },
        };
      }
      return { data: { groups: [] } };
    });

    renderWithProviders(client);

    await user.type(screen.getByLabelText("Group name"), "New Group");
    await user.click(
      screen.getByRole("button", { name: "Create group" }),
    );

    expect(mutationSpy).toHaveBeenCalled();
  });

  it("shows user info and sign out button", () => {
    const client = createTestClient(emptyGroupsHandler);
    renderWithProviders(client);

    expect(screen.getByText("Test User")).toBeInTheDocument();
    expect(screen.getByText("Sign out")).toBeInTheDocument();
  });
});
