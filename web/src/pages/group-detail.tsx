import { useState } from "react";
import { Link, useParams } from "react-router";
import { useMutation, useQuery } from "urql";
import {
  AddMemberToGroupMutation,
  CreateTransactionMutation,
  GroupQuery,
} from "../graphql/operations";
import { useAuth } from "../auth/auth-context";

type GroupMember = {
  id: string;
  email: string;
  displayName: string;
};

type GroupForForm = {
  id: string;
  members: ReadonlyArray<GroupMember>;
};

export function GroupDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [result, reexecuteGroup] = useQuery({
    query: GroupQuery,
    variables: { id: id ?? "" },
    pause: !id,
  });

  const [addResult, addMember] = useMutation(AddMemberToGroupMutation);
  const [email, setEmail] = useState("");
  const [addError, setAddError] = useState<string | null>(null);

  async function handleAddMember(e: React.FormEvent) {
    e.preventDefault();
    setAddError(null);
    const res = await addMember({ groupId: id!, email });
    if (res.error) {
      setAddError(res.error.message);
    } else {
      setEmail("");
      reexecuteGroup({ requestPolicy: "network-only" });
    }
  }

  if (result.fetching) {
    return (
      <div style={{ maxWidth: 480, margin: "80px auto" }}>
        <p>Loading...</p>
      </div>
    );
  }

  if (result.error) {
    return (
      <div style={{ maxWidth: 480, margin: "80px auto" }}>
        <Link to="/">&larr; Back to groups</Link>
        <p style={{ color: "red" }}>
          Failed to load group: {result.error.message}
        </p>
      </div>
    );
  }

  const group = result.data?.group;

  if (!group) {
    return (
      <div style={{ maxWidth: 480, margin: "80px auto" }}>
        <Link to="/">&larr; Back to groups</Link>
        <p>Group not found.</p>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 480, margin: "80px auto" }}>
      <Link to="/">&larr; Back to groups</Link>
      <h1>{group.name}</h1>
      <p style={{ color: "#666" }}>
        Created by {group.createdBy.displayName} on{" "}
        {new Date(group.createdAt).toLocaleDateString()}
      </p>

      <h2>Members</h2>
      <ul>
        {group.members.map((member) => (
          <li key={member.id}>
            {member.displayName} ({member.email})
          </li>
        ))}
      </ul>

      <h3>Add Member</h3>
      <form onSubmit={handleAddMember}>
        <div style={{ marginBottom: 12 }}>
          <label htmlFor="member-email">Email address</label>
          <br />
          <input
            id="member-email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            style={{ width: "100%" }}
          />
        </div>
        {addError && <p style={{ color: "red" }}>{addError}</p>}
        <button type="submit" disabled={addResult.fetching}>
          {addResult.fetching ? "Adding..." : "Add member"}
        </button>
      </form>

      <h2>Transactions</h2>
      <h3>Add Transaction</h3>
      <AddTransactionForm
        group={group}
        onCreated={() => reexecuteGroup({ requestPolicy: "network-only" })}
      />

      {group.transactions.length === 0 ? (
        <p style={{ color: "#666" }}>No transactions yet.</p>
      ) : (
        <ul style={{ listStyle: "none", padding: 0 }}>
          {group.transactions.map((txn) => (
            <li
              key={txn.id}
              style={{
                border: "1px solid #ccc",
                borderRadius: 8,
                padding: "12px 16px",
                marginBottom: 8,
              }}
            >
              <strong>{txn.description}</strong>
              <span style={{ color: "#666", marginLeft: 8 }}>
                ${(txn.amount / 100).toFixed(2)}
              </span>
              <br />
              <span style={{ color: "#666" }}>
                Paid by {txn.paidBy.displayName} on{" "}
                {new Date(txn.createdAt).toLocaleDateString()}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function AddTransactionForm({
  group,
  onCreated,
}: {
  group: GroupForForm;
  onCreated: () => void;
}) {
  const { user } = useAuth();
  const [createResult, createTransaction] = useMutation(
    CreateTransactionMutation,
  );

  const initialPayer = () =>
    user?.id && group.members.some((m) => m.id === user.id)
      ? user.id
      : (group.members[0]?.id ?? "");

  const [description, setDescription] = useState("");
  const [amount, setAmount] = useState("");
  const [paidBy, setPaidBy] = useState<string>(initialPayer);
  const [splitBetween, setSplitBetween] = useState<Set<string>>(
    () => new Set(group.members.map((m) => m.id)),
  );
  const [error, setError] = useState<string | null>(null);

  function toggleSplit(id: string) {
    setSplitBetween((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    const trimmedDescription = description.trim();
    if (!trimmedDescription) {
      setError("Description is required.");
      return;
    }
    const dollars = parseFloat(amount);
    if (!Number.isFinite(dollars) || dollars <= 0) {
      setError("Amount must be greater than zero.");
      return;
    }
    if (splitBetween.size === 0) {
      setError("Select at least one member to split between.");
      return;
    }
    if (!paidBy) {
      setError("Select who paid.");
      return;
    }

    const cents = Math.round(dollars * 100);
    const res = await createTransaction({
      input: {
        groupId: group.id,
        description: trimmedDescription,
        amount: cents,
        paidBy,
        splitBetween: Array.from(splitBetween),
      },
    });

    if (res.error) {
      setError(res.error.message);
      return;
    }

    setDescription("");
    setAmount("");
    setPaidBy(initialPayer());
    setSplitBetween(new Set(group.members.map((m) => m.id)));
    onCreated();
  }

  return (
    <form onSubmit={handleSubmit}>
      <div style={{ marginBottom: 12 }}>
        <label htmlFor="txn-description">Description</label>
        <br />
        <input
          id="txn-description"
          type="text"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          required
          style={{ width: "100%" }}
        />
      </div>
      <div style={{ marginBottom: 12 }}>
        <label htmlFor="txn-amount">Amount</label>
        <br />
        <input
          id="txn-amount"
          type="number"
          step="0.01"
          min="0"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          required
          style={{ width: "100%" }}
        />
      </div>
      <div style={{ marginBottom: 12 }}>
        <label htmlFor="txn-paid-by">Paid by</label>
        <br />
        <select
          id="txn-paid-by"
          value={paidBy}
          onChange={(e) => setPaidBy(e.target.value)}
          style={{ width: "100%" }}
        >
          {group.members.map((m) => (
            <option key={m.id} value={m.id}>
              {m.displayName}
            </option>
          ))}
        </select>
      </div>
      <fieldset
        style={{ marginBottom: 12, border: "1px solid #ccc", padding: 8 }}
      >
        <legend>Split between</legend>
        {group.members.map((m) => (
          <label key={m.id} style={{ display: "block" }}>
            <input
              type="checkbox"
              checked={splitBetween.has(m.id)}
              onChange={() => toggleSplit(m.id)}
            />{" "}
            {m.displayName}
          </label>
        ))}
      </fieldset>
      {error && <p style={{ color: "red" }}>{error}</p>}
      <button type="submit" disabled={createResult.fetching}>
        {createResult.fetching ? "Adding..." : "Add transaction"}
      </button>
    </form>
  );
}
