import { useState } from "react";
import { Link } from "react-router";
import { useMutation, useQuery } from "urql";
import { useAuth } from "../auth/auth-context";
import { CreateGroupMutation, GroupsQuery } from "../graphql/operations";

export function HomePage() {
  const { user, logout } = useAuth();
  const [groupsResult, reexecuteGroups] = useQuery({ query: GroupsQuery });
  const [, createGroup] = useMutation(CreateGroupMutation);

  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleCreateGroup(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const result = await createGroup({ input: { name } });
      if (result.error) {
        throw new Error(result.error.message);
      }
      setName("");
      reexecuteGroups({ requestPolicy: "network-only" });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create group");
    } finally {
      setSubmitting(false);
    }
  }

  const groups = groupsResult.data?.groups;

  return (
    <div style={{ maxWidth: 480, margin: "80px auto" }}>
      <h1>Splitty</h1>
      {user && (
        <p>
          Signed in as <strong>{user.displayName}</strong> ({user.email})
        </p>
      )}

      <h2>Your Groups</h2>
      {groupsResult.fetching ? (
        <p>Loading groups...</p>
      ) : groupsResult.error ? (
        <p style={{ color: "red" }}>
          Failed to load groups: {groupsResult.error.message}
        </p>
      ) : groups && groups.length > 0 ? (
        <ul style={{ listStyle: "none", padding: 0 }}>
          {groups.map((group) => (
            <li key={group.id} style={{ marginBottom: 8 }}>
              <Link
                to={`/groups/${group.id}`}
                style={{ textDecoration: "none", color: "inherit" }}
              >
                <div
                  style={{
                    border: "1px solid #ccc",
                    borderRadius: 8,
                    padding: "12px 16px",
                  }}
                >
                  <strong>{group.name}</strong>
                  <span style={{ color: "#666", marginLeft: 8 }}>
                    {group.members.length}{" "}
                    {group.members.length === 1 ? "member" : "members"}
                  </span>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      ) : (
        <p style={{ color: "#666" }}>No groups yet. Create one below!</p>
      )}

      <h3>Create a Group</h3>
      <form onSubmit={handleCreateGroup}>
        <div style={{ marginBottom: 12 }}>
          <label htmlFor="group-name">Group name</label>
          <br />
          <input
            id="group-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            style={{ width: "100%" }}
          />
        </div>
        {error && <p style={{ color: "red" }}>{error}</p>}
        <button type="submit" disabled={submitting}>
          {submitting ? "Creating..." : "Create group"}
        </button>
      </form>

      <hr style={{ margin: "24px 0" }} />
      <button onClick={logout}>Sign out</button>
    </div>
  );
}
