#!/usr/bin/env bun
import { render } from "ink";
import { Daemira } from "../src/Daemira";
import { cliToState, runDynamicApp } from "@core/app";
import { AppCli } from "./cli";

const instance = new Daemira();
const { returnOutput } = cliToState(instance.getState());

await runDynamicApp(instance);

// Only render the UI if --return is NOT present
if (!returnOutput) {
  render(<AppCli app={instance} />);
}
