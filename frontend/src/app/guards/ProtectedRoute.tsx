import { Navigate } from "react-router-dom";
import { useAuthStore } from "../../core/auth/auth-store";

type Props = { children: JSX.Element };

export function ProtectedRoute({ children }: Props): JSX.Element {
  const auth = useAuthStore();
  if (!auth.isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  return children;
}

