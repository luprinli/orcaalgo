import { useCallback, useEffect, useSyncExternalStore } from "react";
import { useTheme } from "next-themes";

/**
 * Binary light/dark theme toggle.
 * Renders a half-filled circle SVG that inverts on toggle.
 * Uses useSyncExternalStore to avoid hydration mismatch.
 */
function ContrastIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      className={className}
    >
      <path
        fillRule="evenodd"
        d="M10 2a8 8 0 100 16V2zm0 0a8 8 0 010 16V2z"
        clipRule="evenodd"
      />
    </svg>
  );
}

const emptySubscribe = () => () => {};

export function ThemeToggle({ className }: { className?: string }) {
  const { resolvedTheme, setTheme } = useTheme();
  // Prevent hydration mismatch by only rendering after mount
  const mounted = useSyncExternalStore(
    emptySubscribe,
    () => true,
    () => false,
  );

  const toggle = useCallback(() => {
    setTheme(resolvedTheme === "dark" ? "light" : "dark");
  }, [resolvedTheme, setTheme]);

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "T" && e.shiftKey && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        toggle();
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [toggle]);

  if (!mounted) {
    return (
      <button
        className={`inline-flex items-center justify-center rounded-full transition-colors hover:bg-muted size-8 ${className ?? ""}`}
        aria-label="Toggle theme"
        disabled
      >
        <ContrastIcon className="size-4 text-muted-foreground" />
      </button>
    );
  }

  return (
    <button
      onClick={toggle}
      className={`inline-flex items-center justify-center rounded-full transition-colors hover:bg-muted size-8 ${className ?? ""}`}
      aria-label={resolvedTheme === "dark" ? "Switch to light mode" : "Switch to dark mode"}
      title={resolvedTheme === "dark" ? "Light mode" : "Dark mode"}
    >
      <ContrastIcon className="size-4 text-foreground" />
    </button>
  );
}
