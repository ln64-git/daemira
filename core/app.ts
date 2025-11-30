import { isEqual } from "lodash";
import net from "net";

// ═══════════════════════════════════════════════════════════════════════════
// TYPES
// ═══════════════════════════════════════════════════════════════════════════

type JSONValue = string | number | boolean | null | JSONValue[] | { [key: string]: JSONValue };

// ═══════════════════════════════════════════════════════════════════════════
// APP BASE CLASS
// ═══════════════════════════════════════════════════════════════════════════

export abstract class App {
  port = 2000;

  // Serialize instance to JSON (exclude methods)
  toJSON(): Record<string, JSONValue> {
    return Object.fromEntries(
      Object.entries(this).filter(([_, v]) => typeof v !== 'function')
    ) as Record<string, JSONValue>;
  }

  // Deserialize JSON into instance
  fromJSON(data: Record<string, JSONValue>): void {
    Object.entries(data).forEach(([key, value]) => {
      (this as any)[key] = value;
    });
  }

  // Save state to file
  async save(): Promise<void> {
    try {
      await Bun.write('.app-state.json', JSON.stringify(this.toJSON(), null, 2));
    } catch (error) {
      console.error('Failed to save state:', error);
    }
  }

  // Load state from file
  async load(): Promise<void> {
    try {
      const data = await Bun.file('.app-state.json').text();
      this.fromJSON(JSON.parse(data));
    } catch (error) {
      // File doesn't exist yet - that's fine
    }
  }

  // Check if server is running on port
  async probe(): Promise<boolean> {
    try {
      const res = await fetch(`http://localhost:${this.port}/health`);
      return res.ok;
    } catch {
      return false;
    }
  }

  // Set state (local or remote)
  async setState(updates: Record<string, JSONValue>): Promise<Record<string, JSONValue> | undefined> {
    const isLocal = !(await this.probe());

    if (isLocal) {
      this.fromJSON(updates);
      await this.save();
      return this.toJSON();
    }

    try {
      const res = await fetch(`http://localhost:${this.port}/state`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates),
      });
      const response = await res.json() as { state?: Record<string, JSONValue> };
      return response?.state;
    } catch (error) {
      console.error('Failed to set state:', error);
      return undefined;
    }
  }
}

// ═══════════════════════════════════════════════════════════════════════════
// PROXY WRAPPER (AUTO-SAVE)
// ═══════════════════════════════════════════════════════════════════════════

function createStatefulProxy<T extends App>(instance: T, dev: boolean = false): T {
  return new Proxy(instance, {
    set(target, key, value) {
      const result = Reflect.set(target, key, value);

      // Auto-save on property changes (but not methods)
      if (typeof value !== 'function' && key !== 'constructor') {
        target.save().catch(console.error);
      }

      return result;
    },

    get(target, key) {
      const value = Reflect.get(target, key);

      // Optional: Dev mode logging for method calls
      if (dev && typeof value === 'function' && key !== 'constructor') {
        return function (this: any, ...args: any[]) {
          console.time(`[${String(key)}]`);
          const result = value.apply(this, args);

          if (result instanceof Promise) {
            return result.finally(() => console.timeEnd(`[${String(key)}]`));
          }

          console.timeEnd(`[${String(key)}]`);
          return result;
        };
      }

      return value;
    }
  });
}

// ═══════════════════════════════════════════════════════════════════════════
// SYSTEM NOTIFICATION
// ═══════════════════════════════════════════════════════════════════════════

async function sendNotification(title: string, message: string): Promise<void> {
  try {
    // Try notify-send (Linux)
    const result = await Bun.spawn(['notify-send', title, message], {
      stdout: 'ignore',
      stderr: 'ignore',
    }).exited;

    if (result !== 0) {
      // Fallback: just log if notify-send fails
      console.log(`[NOTIFY] ${title}: ${message}`);
    }
  } catch (error) {
    // notify-send not available, just log
    console.log(`[NOTIFY] ${title}: ${message}`);
  }
}

// ═══════════════════════════════════════════════════════════════════════════
// CLI PARSER
// ═══════════════════════════════════════════════════════════════════════════

function parseCLI(): { serve: boolean; port?: number; dev: boolean; notify: boolean; method?: string; methodArgs: any[] } {
  const args = process.argv.slice(2);
  const flags = {
    serve: args.includes('--serve'),
    port: undefined as number | undefined,
    dev: args.includes('--dev'),
    notify: args.includes('--notify'),
    method: undefined as string | undefined,
    methodArgs: [] as any[],
  };

  const portIndex = args.indexOf('--port');
  if (portIndex !== -1 && args[portIndex + 1]) {
    flags.port = parseInt(args[portIndex + 1]!, 10);
  }

  // Find first non-flag argument as method name
  const portValue = portIndex !== -1 ? args[portIndex + 1] : undefined;
  const nonFlagArgs = args.filter(arg => !arg.startsWith('--') && arg !== portValue);

  if (nonFlagArgs.length > 0) {
    flags.method = nonFlagArgs[0];
    // Parse remaining arguments as JSON if possible, otherwise keep as strings
    flags.methodArgs = nonFlagArgs.slice(1).map(arg => {
      try {
        return JSON.parse(arg);
      } catch {
        return arg;
      }
    });
  }

  return flags;
}

// ═══════════════════════════════════════════════════════════════════════════
// SERVER
// ═══════════════════════════════════════════════════════════════════════════

async function findAvailablePort(start: number): Promise<number> {
  for (let port = start; port < start + 50; port++) {
    const isFree = await new Promise<boolean>(resolve => {
      const server: net.Server = net.createServer()
        .once('error', () => resolve(false))
        .once('listening', () => server.close(() => resolve(true)))
        .listen(port);
    });
    if (isFree) return port;
  }
  throw new Error(`No available ports found starting from ${start}`);
}

async function serve<T extends App>(app: T, requestedPort: number): Promise<void> {
  // Check if port is already in use
  const isRunning = await app.probe();
  if (isRunning) {
    console.log(`[${requestedPort}] Server already running, finding next available port...`);
  }

  const port = await findAvailablePort(requestedPort);
  app.port = port;

  const server = Bun.serve({
    port,
    async fetch(req) {
      const url = new URL(req.url);

      // Health check
      if (url.pathname === '/health') {
        return Response.json({ ok: true });
      }

      // State endpoints
      if (url.pathname === '/state') {
        if (req.method === 'GET') {
          return Response.json(app.toJSON());
        }

        if (req.method === 'POST') {
          const body = await req.json() as Record<string, JSONValue>;

          // Calculate diff
          const current = app.toJSON();
          const patch: Record<string, JSONValue> = {};
          for (const key in body) {
            if (!isEqual(body[key], current[key])) {
              const value = body[key];
              if (value !== undefined) {
                patch[key] = value;
              }
            }
          }

          // Apply updates
          if (Object.keys(patch).length > 0) {
            app.fromJSON(patch);
            await app.save();
          }

          return Response.json({ ok: true, state: app.toJSON() });
        }
      }

      // JSON-RPC method dispatch
      if (req.method === 'POST') {
        try {
          const { method, params = [] } = await req.json() as { method: string; params?: any[] };

          if (typeof (app as any)[method] === 'function') {
            const result = await (app as any)[method](...params);
            return Response.json({ result });
          }

          return Response.json({ error: 'Method not found' }, { status: 404 });
        } catch (error: any) {
          return Response.json({ error: error.message }, { status: 500 });
        }
      }

      return new Response('Not Found', { status: 404 });
    }
  });

  console.log(`[${port}] Server started`);

  // Graceful shutdown
  const shutdown = () => {
    server.stop();
    process.exit(0);
  };

  process.on('SIGTERM', shutdown);
  process.on('SIGINT', shutdown);
}

// ═══════════════════════════════════════════════════════════════════════════
// MAIN RUNNER
// ═══════════════════════════════════════════════════════════════════════════

export async function run<T extends App>(app: T): Promise<void> {
  const { serve: shouldServe, port, dev, notify, method, methodArgs } = parseCLI();

  // Set port if provided
  if (port) app.port = port;

  // Load existing state
  await app.load();

  // Wrap in auto-save proxy
  const statefulApp = createStatefulProxy(app, dev);

  // Server mode
  if (shouldServe) {
    await serve(statefulApp, app.port);
    return;
  }

  // Local mode - run specified method or defaultFunction
  const functionToCall = method || 'defaultFunction';

  if (typeof (statefulApp as any)[functionToCall] === 'function') {
    try {
      const result = await (statefulApp as any)[functionToCall](...methodArgs);
      if (result !== undefined) {
        console.log(result);

        // Send notification if --notify flag is set
        if (notify) {
          const message = typeof result === 'string' ? result : JSON.stringify(result);
          await sendNotification(functionToCall, message);
        }
      }
    } catch (error: any) {
      console.error('Error:', error.message);

      // Send error notification if --notify flag is set
      if (notify) {
        await sendNotification(`Error: ${functionToCall}`, error.message);
      }

      process.exit(1);
    }
  } else {
    console.error(`Error: Method '${functionToCall}' not found`);
    process.exit(1);
  }
}
