import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { CombinedError, Provider } from "urql";
import { describe, expect, it, vi } from "vitest";
import { GroupDetailPage } from "./group-detail";
import { createTestClient } from "../test-helpers";

vi.mock("../auth/auth-context", () => ({
  useAuth: () => ({
    user: { id: "1", email: "test@test.com", displayName: "Test User" },
    logout: vi.fn(),
    isAuthenticated: true,
    loading: false,
    login: vi.fn(),
  }),
}));

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

const groupData = {
  __typename: "Group" as const,
  id: "g1",
  name: "Trip to Paris",
  createdAt: "2026-01-15T00:00:00Z",
  createdBy: {
    __typename: "User" as const,
    id: "1",
    displayName: "Test User",
  },
  members: [
    {
      __typename: "User" as const,
      id: "1",
      email: "test@test.com",
      displayName: "Test User",
    },
    {
      __typename: "User" as const,
      id: "2",
      email: "other@test.com",
      displayName: "Other User",
    },
  ],
  transactions: [] as unknown[],
};

const groupHandler = () => ({ data: { group: groupData } });

describe("GroupDetailPage", () => {
  it("renders group details", () => {
    const client = createTestClient(groupHandler);
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
    const client = createTestClient(() => ({
      data: { group: null },
    }));
    renderAtRoute(client);

    expect(screen.getByText("Group not found.")).toBeInTheDocument();
  });

  it("shows error message on query failure", () => {
    const client = createTestClient(() => ({
      error: new CombinedError({
        networkError: new Error("Not authorized"),
      }),
    }));
    renderAtRoute(client);

    expect(
      screen.getByText(/Failed to load group/),
    ).toBeInTheDocument();
  });

  it("has a back link to the groups list", () => {
    const client = createTestClient(groupHandler);
    renderAtRoute(client);

    const backLink = screen.getByText("\u2190 Back to groups");
    expect(backLink.closest("a")).toHaveAttribute("href", "/");
  });

  it("shows the add member form", () => {
    const client = createTestClient(groupHandler);
    renderAtRoute(client);

    expect(screen.getByLabelText("Email address")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Add member" }),
    ).toBeInTheDocument();
  });

  it("submits the add member form", async () => {
    const user = userEvent.setup();
    const mutationSpy = vi.fn();
    const newMember = {
      __typename: "User" as const,
      id: "3",
      email: "new@test.com",
      displayName: "New User",
    };
    const updatedGroupData = {
      ...groupData,
      members: [...groupData.members, newMember],
    };
    let mutated = false;
    const client = createTestClient((op) => {
      if (op.kind === "mutation") {
        mutationSpy(op);
        mutated = true;
        return {
          data: {
            addMemberToGroup: {
              __typename: "Group" as const,
              id: "g1",
              members: updatedGroupData.members,
            },
          },
        };
      }
      return { data: { group: mutated ? updatedGroupData : groupData } };
    });

    renderAtRoute(client);

    await user.type(screen.getByLabelText("Email address"), "new@test.com");
    await user.click(screen.getByRole("button", { name: "Add member" }));

    expect(mutationSpy).toHaveBeenCalled();
    expect(await screen.findByText("New User (new@test.com)")).toBeInTheDocument();
  });

  it("shows error when add member fails", async () => {
    const user = userEvent.setup();
    const client = createTestClient((op) => {
      if (op.kind === "mutation") {
        return {
          error: new CombinedError({
            graphQLErrors: [{ message: "no user with that email" }],
          }),
        };
      }
      return { data: { group: groupData } };
    });

    renderAtRoute(client);

    await user.type(screen.getByLabelText("Email address"), "nobody@test.com");
    await user.click(screen.getByRole("button", { name: "Add member" }));

    expect(
      await screen.findByText(/no user with that email/),
    ).toBeInTheDocument();
  });

  it("shows empty state for no transactions", () => {
    const client = createTestClient(groupHandler);
    renderAtRoute(client);

    expect(screen.getByText("No transactions yet.")).toBeInTheDocument();
  });

  it("renders the add transaction form with current user as default payer and all members checked", () => {
    const client = createTestClient(groupHandler);
    renderAtRoute(client);

    expect(screen.getByLabelText("Description")).toBeInTheDocument();
    expect(screen.getByLabelText("Amount")).toBeInTheDocument();

    const paidBy = screen.getByLabelText("Paid by") as HTMLSelectElement;
    expect(paidBy.value).toBe("1");

    const testUserCheckbox = screen.getByRole("checkbox", {
      name: /Test User/,
    }) as HTMLInputElement;
    const otherUserCheckbox = screen.getByRole("checkbox", {
      name: /Other User/,
    }) as HTMLInputElement;
    expect(testUserCheckbox.checked).toBe(true);
    expect(otherUserCheckbox.checked).toBe(true);
  });

  function makeCreateTxnClient(onMutation?: (op: unknown) => void) {
    return createTestClient((op) => {
      if (op.kind === "mutation") {
        onMutation?.(op);
        return {
          data: {
            createTransaction: {
              __typename: "Transaction" as const,
              id: "t-new",
            },
          },
        };
      }
      return { data: { group: groupData } };
    });
  }

  it("submits createTransaction with the right input", async () => {
    const user = userEvent.setup();
    const mutationSpy = vi.fn();
    renderAtRoute(makeCreateTxnClient(mutationSpy));

    await user.type(screen.getByLabelText("Description"), "Dinner");
    await user.type(screen.getByLabelText("Amount"), "42.50");
    await user.click(screen.getByRole("button", { name: "Add transaction" }));

    expect(mutationSpy).toHaveBeenCalled();
    const op = mutationSpy.mock.calls[0]![0];
    expect(op.variables.input).toEqual({
      groupId: "g1",
      description: "Dinner",
      amount: 4250,
      paidBy: "1",
      splitBetween: ["1", "2"],
    });
  });

  it("allows changing the payer", async () => {
    const user = userEvent.setup();
    const mutationSpy = vi.fn();
    renderAtRoute(makeCreateTxnClient(mutationSpy));

    await user.type(screen.getByLabelText("Description"), "Cab");
    await user.type(screen.getByLabelText("Amount"), "12");
    await user.selectOptions(screen.getByLabelText("Paid by"), "2");
    await user.click(screen.getByRole("button", { name: "Add transaction" }));

    const op = mutationSpy.mock.calls[0]![0];
    expect(op.variables.input.paidBy).toBe("2");
  });

  it("excludes unchecked members from splitBetween", async () => {
    const user = userEvent.setup();
    const mutationSpy = vi.fn();
    renderAtRoute(makeCreateTxnClient(mutationSpy));

    await user.type(screen.getByLabelText("Description"), "Solo coffee");
    await user.type(screen.getByLabelText("Amount"), "5");
    await user.click(screen.getByRole("checkbox", { name: /Other User/ }));
    await user.click(screen.getByRole("button", { name: "Add transaction" }));

    const op = mutationSpy.mock.calls[0]![0];
    expect(op.variables.input.splitBetween).toEqual(["1"]);
  });

  it("shows the server error when createTransaction fails", async () => {
    const user = userEvent.setup();
    const client = createTestClient((op) => {
      if (op.kind === "mutation") {
        return {
          error: new CombinedError({
            graphQLErrors: [{ message: "amount must be positive" }],
          }),
        };
      }
      return { data: { group: groupData } };
    });

    renderAtRoute(client);

    await user.type(screen.getByLabelText("Description"), "Bad");
    await user.type(screen.getByLabelText("Amount"), "10");
    await user.click(screen.getByRole("button", { name: "Add transaction" }));

    expect(
      await screen.findByText(/amount must be positive/),
    ).toBeInTheDocument();
  });

  it("renders transaction list", () => {
    const groupWithTransactions = {
      ...groupData,
      transactions: [
        {
          __typename: "Transaction" as const,
          id: "t1",
          description: "Dinner",
          amount: 5000,
          paidBy: {
            __typename: "User" as const,
            id: "1",
            displayName: "Test User",
          },
          createdAt: "2026-01-16T00:00:00Z",
        },
        {
          __typename: "Transaction" as const,
          id: "t2",
          description: "Hotel",
          amount: 20000,
          paidBy: {
            __typename: "User" as const,
            id: "2",
            displayName: "Other User",
          },
          createdAt: "2026-01-17T00:00:00Z",
        },
      ],
    };

    const client = createTestClient(() => ({
      data: { group: groupWithTransactions },
    }));
    renderAtRoute(client);

    expect(screen.getByText("Dinner")).toBeInTheDocument();
    expect(screen.getByText("$50.00")).toBeInTheDocument();
    expect(screen.getByText(/Paid by Test User/)).toBeInTheDocument();
    expect(screen.getByText("Hotel")).toBeInTheDocument();
    expect(screen.getByText("$200.00")).toBeInTheDocument();
    expect(screen.getByText(/Paid by Other User/)).toBeInTheDocument();
  });
});
