import { useEffect, useState } from "react";
import { Box, Text, useApp, useInput } from "ink";
import TextInput from "ink-text-input";
import type { DynamicServerApp } from "./app";

export interface AppProps {
  app: DynamicServerApp<any>;
}

export function AppCli({ app }: AppProps) {
  const { exit } = useApp();
  const [state, setState] = useState<Record<string, any>>({});
  const [inputValue, setInputValue] = useState("");
  const [logMessage, setLogMessage] = useState<string | null>(null);
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState<number | null>(null);
  const [isSynced, setIsSynced] = useState<boolean>(false);
  const [cursorVisible, setCursorVisible] = useState(true);

  useEffect(() => {
    const cursorInterval = setInterval(() => {
      setCursorVisible((v) => !v);
    }, 500);
    return () => clearInterval(cursorInterval);
  }, []);

  async function refresh() {
    const isRunning = await app.probe();
    const data = isRunning
      ? await fetch(`http://localhost:${app.port}/state`).then((r) => r.json())
      : app.getState();
    setState({ ...(typeof data === "object" && data !== null ? data : {}) });
  }

  useEffect(() => {
    let interval: NodeJS.Timeout;
    const initialize = async () => {
      const schemaKeys = Object.keys(app.schema.shape);
      setState((prev) => {
        const placeholder: Record<string, any> = {};
        for (const key of schemaKeys) {
          placeholder[key] = prev[key] ?? "";
        }
        return placeholder;
      });

      const isRunning = await app.probe();
      setIsSynced(isRunning);

      await refresh();
      interval = setInterval(refresh, 1000);
    };
    void initialize();
    return () => clearInterval(interval);
  }, []);

  const functionNames = Object.getOwnPropertyNames(Object.getPrototypeOf(app))
    .filter((k) => typeof (app as any)[k] === "function" && k !== "constructor");

  async function handleCommand(command: string) {
    const [cmd, ...args] = command.trim().split(" ");

    if (cmd === "exit") return exit();

    if (cmd === "get") {
      const key = args[0];
      if (!key) return setLogMessage("Please specify a key.");
      if (key === "port") return setLogMessage(`Access to 'port' is restricted.`);
      setLogMessage(`${state[key as keyof typeof state] ?? ""}`);
    }

    else if (cmd === "set") {
      const key = String(args[0]);
      if (key === "port") return setLogMessage(`'port' cannot be modified.`);
      const value = args.slice(1).join(" ");
      const updatedState = await app.set({ [key]: value });
      if (updatedState) {
        setState({ ...updatedState });
      } else {
        await refresh();
      }
      setLogMessage(`set ${key}: ${value}`);
    }

    else if (cmd === "call" || cmd?.endsWith("()")) {
      const fn = cmd === "call" ? args[0] : cmd.slice(0, -2);
      if (!fn) return setLogMessage("Please specify a function to call.");
      try {
        const isRunning = await app.probe();
        const result = isRunning
          ? await fetch(`http://localhost:${app.port}/${fn}`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify([]),
          }).then((r) => r.json())
          : await (app as any)[fn]();
        const output = isRunning ? result.result : result;
        await refresh();
        setLogMessage(typeof output === "string" ? output : JSON.stringify(output, null, 2));
      } catch (e: any) {
        setLogMessage(`Error: ${e.message}`);
      }
    }

    else {
      setLogMessage(`Unknown command: ${command}`);
    }

    setHistory((prev) => [...prev, command]);
    setHistoryIndex(null);
    setInputValue("");
  }

  const className = app.constructor.name;

  function renderTypedInput() {
    const [first, ...rest] = inputValue.trim().split(" ");
    const key = rest[0];
    const valueText = rest.slice(1).join(" ");
    const isFn = inputValue.trim().endsWith("()");
    const isGetSet = first === "get" || first === "set";
    const isCall = first === "call";
    const isKnownVariable = key && Object.prototype.hasOwnProperty.call(state, key);

    if (isGetSet || isCall) {
      return (
        <Text>
          <Text color={isCall ? "yellow" : "magenta"}>{first}</Text>
          {key && (
            <>
              <Text> </Text>
              <Text color={isKnownVariable ? "gray" : "cyan"}>{key}</Text>
              {valueText && (
                <>
                  <Text> </Text>
                  <Text color="white">{valueText}</Text>
                </>
              )}
            </>
          )}
        </Text>
      );
    }

    if (isFn) return <Text color="blue">{inputValue}</Text>;

    return <Text color="cyan">{inputValue}</Text>;
  }

  useInput((input, key) => {
    if (key.upArrow && history.length) {
      setHistoryIndex((prev) => {
        const newIndex = prev === null ? history.length - 1 : Math.max(0, prev - 1);
        setInputValue(history[newIndex] || "");
        return newIndex;
      });
    }
    if (key.downArrow && history.length) {
      setHistoryIndex((prev) => {
        if (prev === null) return null;
        const newIndex = prev + 1;
        if (newIndex >= history.length) {
          setInputValue("");
          return null;
        }
        setInputValue(history[newIndex] || "");
        return newIndex;
      });
    }
  });

  return (
    <Box flexDirection="column" paddingLeft={2}>
      <Text>
        <Text color="cyan" bold>{className}</Text>
        <Text color="gray"> (port {app.port})</Text>
        {isSynced && !app.isServerInstance && <Text color="red"> (Remote)</Text>}
      </Text>

      {/* Conditionally show Variables */}
      {Object.keys(state).some((key) => key !== "port" && key !== "isServerInstance") && (
        <>
          <Text bold>Variables:</Text>
          <Box flexDirection="column" paddingLeft={2}>
            {Object.entries(state)
              .filter(([key]) => key !== "port" && key !== "isServerInstance")
              .map(([key, val]) => (
                <Text key={key}>
                  <Text color="gray">{key.padEnd(12)}</Text>
                  <Text color={val === "…" ? "gray" : "white"} italic={val === "…"}>
                    {String(val)}
                  </Text>
                </Text>
              ))}
          </Box>
        </>
      )}

      {/* Conditionally show Functions */}
      {functionNames.length > 0 && (
        <>
          <Text bold>Functions:</Text>
          <Box flexDirection="column" paddingLeft={2}>
            {functionNames.map((fn) => (
              <Text key={fn}>
                <Text color="blue">{fn}()</Text>
              </Text>
            ))}
          </Box>
        </>
      )}

      <Box paddingTop={1}>
        <Text color="white"> {logMessage}</Text>
      </Box>

      <Box>
        <Text color="cyan">▸ </Text>
        {renderTypedInput()}
        {cursorVisible && <Text color="cyan">│</Text>}
      </Box>

      <Box height={0}>
        <TextInput
          value={inputValue}
          onChange={setInputValue}
          onSubmit={handleCommand}
        />
      </Box>
    </Box>
  );

}
