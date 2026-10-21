/* @refresh reload */
// Entry point: mounts <App/> into #root and loads the global stylesheet.
import { render } from "solid-js/web";
import "./index.css";
import App from "./App.tsx";

const root = document.getElementById("root");

render(() => <App />, root!);
