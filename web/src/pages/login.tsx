import { useState } from "react";
import { useNavigate } from "react-router";
import { useMutation } from "urql";
import { useAuth } from "../auth/auth-context";
import { SendPasscodeMutation } from "../graphql/operations";

type Step = "email" | "code";

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [step, setStep] = useState<Step>("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [, sendPasscode] = useMutation(SendPasscodeMutation);

  async function handleSendPasscode(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const result = await sendPasscode({ email });
      if (result.error) {
        throw new Error(result.error.message);
      }
      setStep("code");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to send code");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleVerifyPasscode(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await login(email, code);
      navigate("/", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div style={{ maxWidth: 320, margin: "80px auto" }}>
      <h1>Splitty</h1>
      {step === "email" ? (
        <form onSubmit={handleSendPasscode}>
          <div style={{ marginBottom: 12 }}>
            <label htmlFor="email">Email</label>
            <br />
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoFocus
              style={{ width: "100%" }}
            />
          </div>
          {error && <p style={{ color: "red" }}>{error}</p>}
          <button type="submit" disabled={submitting}>
            {submitting ? "Sending..." : "Send code"}
          </button>
        </form>
      ) : (
        <form onSubmit={handleVerifyPasscode}>
          <p>
            Code sent to <strong>{email}</strong>
          </p>
          <div style={{ marginBottom: 12 }}>
            <label htmlFor="code">Passcode</label>
            <br />
            <input
              id="code"
              type="text"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              required
              autoFocus
              style={{ width: "100%" }}
            />
          </div>
          {error && <p style={{ color: "red" }}>{error}</p>}
          <button type="submit" disabled={submitting}>
            {submitting ? "Signing in..." : "Sign in"}
          </button>
          <br />
          <button
            type="button"
            onClick={() => {
              setStep("email");
              setCode("");
              setError(null);
            }}
            style={{
              background: "none",
              border: "none",
              color: "#666",
              cursor: "pointer",
              padding: "8px 0",
              textDecoration: "underline",
            }}
          >
            Use a different email
          </button>
        </form>
      )}
    </div>
  );
}
