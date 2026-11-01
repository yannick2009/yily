// Root router. A boot gate determines auth state first, then routes into the
// first-run setup, the login page, or the authenticated shell.
import { createResource, Show } from "solid-js";
import { Navigate, Route, Router, type RouteSectionProps } from "@solidjs/router";
import { AppShell } from "./layouts/AppShell";
import { Spinner } from "./components/ui";
import { AuditRoute } from "./routes/AuditRoute";
import { DashboardRoute } from "./routes/DashboardRoute";
import { EnvironmentRoute } from "./routes/EnvironmentRoute";
import { InitRoute } from "./routes/InitRoute";
import { LoginRoute } from "./routes/LoginRoute";
import { ProjectDetailRoute } from "./routes/ProjectDetailRoute";
import { ProfileRoute } from "./routes/ProfileRoute";
import { ProjectsRoute } from "./routes/ProjectsRoute";
import { SettingsRoute } from "./routes/SettingsRoute";
import { TokensRoute } from "./routes/TokensRoute";
import { VariableDetailRoute } from "./routes/VariableDetailRoute";
import { boot, currentUser, initialized } from "./store";

// Splash shown while the auth state is being determined.
function Splash() {
  return (
    <div class="flex min-h-screen items-center justify-center bg-bg text-ink">
      <Spinner size={28} />
    </div>
  );
}

// ProtectedShell guards every route inside the app shell.
function ProtectedShell(props: RouteSectionProps) {
  if (!initialized()) return <Navigate href="/init" />;
  if (!currentUser()) return <Navigate href="/login" />;
  return <AppShell {...props} />;
}

// InitOnly is valid only before an account exists.
function InitOnly() {
  if (initialized()) return <Navigate href={currentUser() ? "/" : "/login"} />;
  return <InitRoute />;
}

// LoginOnly is valid only once initialized but not signed in.
function LoginOnly() {
  if (!initialized()) return <Navigate href="/init" />;
  if (currentUser()) return <Navigate href="/" />;
  return <LoginRoute />;
}

export default function App() {
  const [ready] = createResource(boot);

  return (
    <Show when={!ready.loading} fallback={<Splash />}>
      <Router>
        <Route path="/init" component={InitOnly} />
        <Route path="/login" component={LoginOnly} />
        <Route path="/" component={ProtectedShell}>
          <Route path="/" component={DashboardRoute} />
          <Route path="/projects" component={ProjectsRoute} />
          <Route path="/tokens" component={TokensRoute} />
          <Route path="/audit" component={AuditRoute} />
          <Route path="/profile" component={ProfileRoute} />
          <Route path="/settings" component={SettingsRoute} />
          <Route path="/projects/:projectId" component={ProjectDetailRoute} />
          <Route path="/projects/:projectId/environments/:envId" component={EnvironmentRoute} />
          <Route
            path="/projects/:projectId/environments/:envId/variables/:variableId"
            component={VariableDetailRoute}
          />
        </Route>
      </Router>
    </Show>
  );
}
