// Root router: /login sits outside the shell, every other page renders inside it.
import { Navigate, Route, Router } from "@solidjs/router";
import { AppShell } from "./layouts/AppShell";
import { EnvironmentRoute } from "./routes/EnvironmentRoute";
import { LoginRoute } from "./routes/LoginRoute";
import { ProjectDetailRoute } from "./routes/ProjectDetailRoute";
import { ProjectsRoute } from "./routes/ProjectsRoute";
import { SettingsRoute } from "./routes/SettingsRoute";
import { TokensRoute } from "./routes/TokensRoute";
import { VariableDetailRoute } from "./routes/VariableDetailRoute";

export default function App() {
  return (
    <Router>
      <Route path="/login" component={LoginRoute} />
      <Route path="/" component={AppShell}>
        <Route path="/" component={() => <Navigate href="/projects" />} />
        <Route path="/projects" component={ProjectsRoute} />
        <Route path="/tokens" component={TokensRoute} />
        <Route path="/settings" component={SettingsRoute} />
        <Route path="/projects/:projectId" component={ProjectDetailRoute} />
        <Route path="/projects/:projectId/environments/:envId" component={EnvironmentRoute} />
        <Route
          path="/projects/:projectId/environments/:envId/variables/:variableId"
          component={VariableDetailRoute}
        />
      </Route>
    </Router>
  );
}
