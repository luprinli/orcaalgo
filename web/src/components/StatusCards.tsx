import { Card, CardContent } from "./ui/card";

export function LoadingCard({ message = "Loading..." }: { message?: string }) {
  return (
    <Card>
      <CardContent className="py-6 text-center text-sm text-muted-foreground">
        {message}
      </CardContent>
    </Card>
  );
}

export function ErrorCard({
  error,
  onRetry,
}: {
  error: string;
  onRetry?: () => void;
}) {
  return (
    <Card className="border-l-4 border-l-destructive mb-4">
      <CardContent className="py-3 text-sm text-destructive">
        {error}
        {onRetry && (
          <button
            onClick={onRetry}
            className="ml-3 text-xs underline text-destructive hover:text-destructive/80"
          >
            Retry
          </button>
        )}
      </CardContent>
    </Card>
  );
}

export function SuccessCard({ message }: { message: string }) {
  return (
    <Card className="border-l-4 border-l-trading-success mb-4">
      <CardContent className="py-3 text-sm text-trading-success">{message}</CardContent>
    </Card>
  );
}

export function MessageCard({ message }: { message: string }) {
  const isError = message.toLowerCase().includes("fail") || message.toLowerCase().includes("error");
  return (
    <Card className={`border-l-4 mb-4 ${isError ? "border-l-destructive" : "border-l-trading-success"}`}>
      <CardContent className={`py-3 text-sm ${isError ? "text-destructive" : "text-trading-success"}`}>
        {message}
      </CardContent>
    </Card>
  );
}
