import { Link } from "@tanstack/react-router";

interface Props {
  to: string;
}

export function BackLink({ to }: Props) {
  return (
    <Link
      to={to}
      className="inline-block px-4 py-1.5 border border-border rounded bg-surface text-text cursor-pointer text-sm hover:border-accent mb-4 no-underline"
    >
      ← Back
    </Link>
  );
}
