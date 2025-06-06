import { ZodObject } from "zod";
import { isEqual } from "lodash"; // or write your own deepCompare

export abstract class DynamicServerApp<T extends Record<string, any>> {
  abstract port: number;
  abstract schema: ZodObject<any>;
  isServerInstance: boolean = false; // ← new field
  logToUI: ((message: string) => void) | null = null;

  getState(): Partial<T> {
    const state: Partial<T> = {};

    // own enumerable properties
    for (const key of Object.keys(this)) {
      if (key !== "schema" && typeof (this as any)[key] !== "function") {
        state[key as keyof T] = (this as any)[key];
      }
    }

    // include getter properties from the prototype chain
    let proto = Object.getPrototypeOf(this);
    while (proto && proto !== Object.prototype) {
      for (const key of Object.getOwnPropertyNames(proto)) {
        if (key === "constructor" || key in state) continue;
        const desc = Object.getOwnPropertyDescriptor(proto, key);
        if (desc && typeof desc.get === "function") {
          state[key as keyof T] = (this as any)[key];
        }
      }
      proto = Object.getPrototypeOf(proto);
    }

    return state;
  }

  applyStateUpdate(data: Partial<T>): void {
    const validated = this.schema.partial().parse(data);
    Object.entries(validated).forEach(([key, value]) => {
      if (key in this) {
        (this as any)[key] = value;
      }
    });
  }

  getMetadata(): Record<string, string> {
    return Object.getOwnPropertyNames(this).reduce((meta, key) => {
      const val = (this as any)[key];
      if (typeof val !== "function") meta[key] = typeof val;
      return meta;
    }, {} as Record<string, string>);
  }

  async probe(timeout = 1000): Promise<boolean> {
    try {
      const controller = new AbortController();
      const id = setTimeout(() => controller.abort(), timeout);

      const res = await fetch(`http://localhost:${this.port}/state`, {
        signal: controller.signal,
      });
      clearTimeout(id);
      return res.ok;
    } catch {
      return false;
    }
  }

  async set(diff: Partial<T>): Promise<Partial<T> | undefined> {
    const isLocalServer = await this.probe() === false;

    if (isLocalServer) {
      this.applyStateUpdate(diff);
      return this.getState(); // immediate result
    }

    try {
      const res = await fetch(`http://localhost:${this.port}/state`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(diff),
      });

      const response = await res.json();
      if (response && typeof response === "object" && "state" in response) {
        return (response as { state: Partial<T> }).state;
      }
    } catch (e) {
      console.error("❌ Failed to set state:", e);
    }

    return undefined;
  }


}

export async function runDynamicApp<T extends Record<string, any>>(appInstance: DynamicServerApp<T>): Promise<void> {
  const defaults = appInstance.getState() as T;
  const { command, key, value, returnOutput } = cliToState(defaults);
  const isRunning = await appInstance.probe();

  if (command === "get" && key) {
    const current: Partial<T> = isRunning
      ? await fetch(`http://localhost:${appInstance.port}/state`).then((r) => r.json() as Promise<Partial<T>>)
      : appInstance.getState();

    const output = current[key as keyof T];
    console.log(returnOutput ? output : `${output}`);
    process.exit(0);
  }

  if (command === "set" && key && value !== undefined) {
    const update = { [key]: value } as Partial<T>;
    await appInstance.set(update);
    if (returnOutput) {
      const current = appInstance.getState();
      console.log(current[key as keyof T]);
    }
    process.exit(0);
  }

  if (command === "call" && key) {
    const result = isRunning
      ? await fetch(`http://localhost:${appInstance.port}/${key}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify([]),
      }).then((r) => r.json()).then((res) => (res as { result: any }).result)
      : await (appInstance as any)[key]();

    if (result !== undefined) {
      console.log(returnOutput ? result : `${result}`);
    }
    process.exit(0);
  }

  return startServer(appInstance, { port: appInstance.port, routes: buildRoutes(appInstance) });
}


function buildRoutes<T extends Record<string, any>>(
  appInstance: DynamicServerApp<T>
): Record<string, RemoteAction<T>> {
  return Object.getOwnPropertyNames(Object.getPrototypeOf(appInstance))
    .filter(key => key !== "constructor" && typeof (appInstance as any)[key] === "function")
    .reduce((acc, key) => {
      acc[`/${key}`] = async (app, args) => {
        const actualArgs = Array.isArray(args) ? args : [];
        return await (app as any)[key](...actualArgs);
      };
      return acc;
    }, {} as Record<string, RemoteAction<T>>);
}

export type RemoteAction<T extends Record<string, any>> = (
  appInstance: DynamicServerApp<T>,
  args?: any
) => Promise<any>;

import http from "http";

export async function startServer<T extends Record<string, any>>(
  appInstance: DynamicServerApp<T>,
  options: {
    port?: number;
    routes?: Record<string, RemoteAction<T>>;
  } = {}
) {
  let port = options.port ?? 2001;
  const routes = options.routes ?? {};

  // 🔍 Ensure port is free or find a new one
  port = await findAvailablePort(port);
  appInstance.port = port;

  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url ?? "/", `http://${req.headers.host}`);
    const method = req.method ?? "GET";

    if (url.pathname === "/state") {
      if (method === "GET") {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify(appInstance.getState()));
        return;
      }

      if (method === "POST") {
        let body = "";
        req.on("data", chunk => (body += chunk));
        req.on("end", () => {
          try {
            if (!body) throw new Error("Empty request body");

            const parsed = JSON.parse(body);
            const before = appInstance.getState();
            const patch: Partial<T> = {};

            for (const key in parsed) {
              if (!isEqual(parsed[key], before[key])) {
                (patch as any)[key] = parsed[key];
              }
            }

            if (Object.keys(patch).length > 0) {
              appInstance.applyStateUpdate(patch);
            }

            const newState = appInstance.getState();
            res.writeHead(200, { "Content-Type": "application/json" });
            res.end(JSON.stringify({ status: "ok", state: newState }));
          } catch (err: any) {
            res.writeHead(400, { "Content-Type": "application/json" });
            res.end(JSON.stringify({ error: err.message || "Invalid JSON" }));
          }
        });
        return;
      }

      res.writeHead(405);
      res.end("Method Not Allowed");
      return;
    }

    if (method === "POST" && routes[url.pathname]) {
      let body = "";
      req.on("data", chunk => (body += chunk));
      req.on("end", async () => {
        try {
          const parsed = JSON.parse(body);
          if (typeof routes[url.pathname] === "function") {
            const result = await routes[url.pathname]!(appInstance, parsed);

            if (typeof result === "string") {
              appInstance.logToUI?.(result);
            } else if (result != null) {
              appInstance.logToUI?.(JSON.stringify(result, null, 2));
            }

            res.writeHead(200, { "Content-Type": "application/json" });
            res.end(JSON.stringify({ status: "ok", result }));
          } else {
            res.writeHead(404, { "Content-Type": "application/json" });
            res.end(JSON.stringify({ error: "Route not found" }));
          }
        } catch (err: any) {
          res.writeHead(500, { "Content-Type": "application/json" });
          res.end(JSON.stringify({ error: err.message }));
        }
      });
      return;
    }


    res.writeHead(404);
    res.end("Not Found");
  });

  appInstance.isServerInstance = true;
  server.listen(port);
}


// src/core/CLI.ts
export function cliToState<T extends Record<string, any>>(defaults: T): {
  command: "get" | "set" | "call" | null;
  key?: string;
  value?: string;
  returnOutput: boolean;
} {
  const args = process.argv.slice(2);
  let command: "get" | "set" | "call" | null = null;
  let returnOutput = false;
  let key: string | undefined;
  let value: string | undefined;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (arg === "get" || arg === "set" || arg === "call") {
      command = arg;
      key = args[i + 1];
      value = args[i + 2];
      break;
    }

    if (arg === "--return") {
      returnOutput = true;
    }
  }

  return { command, key, value, returnOutput };
}




export function diffStatePatch<T extends Record<string, any>>(cliArgs: T, currentState: Partial<T>): Partial<T> {
  const patch: Partial<T> = {};
  for (const key in cliArgs) {
    if (cliArgs[key] !== undefined && cliArgs[key] !== currentState[key]) {
      patch[key] = cliArgs[key];
    }
  }
  return patch;
}


import net from "net";

async function findAvailablePort(start: number, maxAttempts = 50): Promise<number> {
  let port = start;
  for (let i = 0; i < maxAttempts; i++) {
    const isFree = await new Promise<boolean>((resolve) => {
      const tester = net.createServer()
        .once("error", () => resolve(false))
        .once("listening", function () {
          tester.close();
          resolve(true);
        })
        .listen(port);
    });

    if (isFree) return port;
    port++;
  }
  throw new Error(`No available ports found starting from ${start}`);
}
