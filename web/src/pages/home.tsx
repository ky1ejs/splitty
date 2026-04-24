import { useAuth } from "../auth/auth-context";

export function HomePage() {
  const { user, logout } = useAuth();

  return (
    <div style={{ maxWidth: 480, margin: "80px auto" }}>
      <h1>Splitty</h1>
      {user ? (
        <p>
          Signed in as <strong>{user.displayName}</strong> ({user.email})
        </p>
      ) : (
        <p>Could not load user.</p>
      )}
      <button onClick={logout}>Sign out</button>
    </div>
  );
}
