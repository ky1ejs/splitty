import { Link, useParams } from "react-router";
import { useQuery } from "urql";
import { GroupQuery } from "../graphql/operations";

export function GroupDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [result] = useQuery({
    query: GroupQuery,
    variables: { id: id ?? "" },
    pause: !id,
  });

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
    </div>
  );
}
