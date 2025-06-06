# 🧙‍♂️ Dynamic Server App Template

A powerful, type-safe, schema-driven dynamic server framework built on **Bun** + **Zod**. Extend this abstract class to build highly customizable backend apps with introspectable state, CLI control, and auto-routed HTTP methods.

## ✨ Features

- 🧠 State introspection & dynamic updates
- 🛡️ Zod-validated schema binding
- ⚙️ Built-in HTTP JSON API (`/state`, `/method`)
- 🧪 Probes for live server detection
- 📟 CLI flags to get/set server state directly
- 🧬 Auto-routing of class methods as endpoints

## 📦 Tech Stack

- [Bun](https://bun.sh)
- [Zod](https://zod.dev)

## 🔧 Usage

### 1. Extend the `DynamicServerApp`

```ts
import { z } from "zod";
import { DynamicServerApp } from "../core/app";

export type SampleState = z.infer<typeof SampleSchema>;
export const SampleSchema = z.object({
  port: z.number(),
});

export class SampleClass extends DynamicServerApp<SampleState> {
  schema = SampleSchema;
  port = 2000;
  message = "Hello, world!";

  async sampleFunction(): Promise<void> {
    console.log(this.message);
  }
}
````

### 2. Run the App

```ts
import { runDynamicApp } from "./app";
import { SampleClass } from "./SampleClass";

runDynamicApp(new SampleClass());
```

## 🖥️ API Endpoints

| Endpoint         | Method | Description                            |
| ---------------- | ------ | -------------------------------------- |
| `/state`         | GET    | Fetch current application state        |
| `/state`         | POST   | Update application state via JSON      |
| `/<method-name>` | POST   | Auto-exposed instance methods via path |

## 🧪 CLI Flags

| Flag          | Description                     |
| ------------- | ------------------------------- |
| `--key value` | Shorthand to set a state value  |
| `-get --key`  | Display the current state value |
| `-set --key`  | Set a state value interactively |

## 🚀 Server Lifecycle

1. If CLI `-get`/`-set` provided → runs as command client.
2. If server not detected → spins up new Bun HTTP server.
3. Auto-routes all non-constructor methods as POST endpoints.

## 🧠 Method Routing Example

```ts
// Call this remotely:
await fetch('/sampleFunction', { method: 'POST' });
```

## 📚 Schema Validation

All state changes and updates are type-checked and validated using the provided `ZodObject` schema. Automatically supports partial updates.

## 🧙‍♂️ CLI to State

CLI state parsing supports dynamic mutation via `--key value` syntax. Use `-get` to fetch specific keys, `-set` to apply.

## 🛠️ Development & Deployment

To run locally:

```bash
bun run index.ts
```

## 🔐 Type Safety

Powered by `z.infer<typeof schema>` — state and routes are always strictly typed.

## 🧩 Extensibility

* Add any methods → automatically exposed as API routes.
* Add more fields to the Zod schema → instantly supported in state.
