import { useState } from "react";
import { Link, useParams } from "react-router";
import { useMutation, useQuery } from "urql";
import {
  AddMemberToGroupMutation,
  GroupQuery,
} from "../graphql/operations";

export function GroupDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [result] = useQuery({
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
