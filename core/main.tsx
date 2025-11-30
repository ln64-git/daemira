#!/usr/bin/env bun
import { Daemira } from "../src/Daemira";
import { run } from "@core/app";

const instance = new Daemira();

await run(instance);
