import { Paper } from "@mui/material";
import React, { useEffect } from "react";
import CodeEditor from "./components/CodeEditor";
import FileView from "./components/FileView";
import MainAppBar from "./components/MainAppBar";
import ResMonitor from "./components/ResMonitor";
import CompView from "./components/CompView";
import Websocket from "react-websocket";
import PoolConfigurator, { pool, updatePool } from "./memory";
import Terminal from "./components/Terminal";
import MainDrawer from "./components/MainDrawer";
import ObjectStore from "./components/ObjectStore";

export let colors;
export let updateColors;

export let currentItemId;
export let setCurrentItemId;

export let forceUpdateApp;

let terminalLines = [];
let terminalBottom;
let terminalIsBusy;
let terminalPrompt;
let terminalInputFocused;

function useForceUpdate() {
  const [value, setValue] = React.useState(0); // integer state
  return () => setValue((value) => ++value); // update the state to force render
}

function App() {
  forceUpdateApp = useForceUpdate();
  const [cpu, setCpu] = React.useState(1);
  const [mem, setMem] = React.useState(1);
  const [fileTree, setFileTree] = React.useState({});
  [currentItemId, setCurrentItemId] = React.useState(undefined);
  [colors, updateColors] = React.useState({
    colorDark1: "#091010",
    colorDark2: "#0e1818",
    colorDark3: "#132020",
    colorAccent: "#2f4f4f",
  });
  const [loaded, setLoaded] = React.useState(false);
  const [drawerOpen, setDrawerOpen] = React.useState(false);
  useEffect(() => {
    window.addEventListener(
      "keydown",
      function (e) {
        if (
          (window.navigator.platform.match("Mac") ? e.metaKey : e.ctrlKey) &&
          e.keyCode == 67
        ) {
          if (terminalInputFocused) {
            e.preventDefault();
            const options = {
              method: "post",
              headers: {
                Accept: "application/json, text/plain, */*",
                "Content-Type": "application/json",
              },
            };
            fetch("../kill-current", options);
          }
        } else if (e.keyCode == 13) {
          let terminalInput = this.document.getElementById("terminalInput");
          let terminalBottom = this.document.getElementById("terminalBottom");
          if (!terminalIsBusy && terminalInputFocused) {
            terminalLines.push({
              type: "me",
              content: terminalPrompt + "$ " + terminalInput.value,
            });
            if (terminalInput.value.startsWith("cd ")) {
              const options = {
                method: "post",
                headers: {
                  Accept: "application/json, text/plain, */*",
                  "Content-Type": "application/json",
                },
                body: JSON.stringify({
                  path: terminalInput.value.substring(3),
                }),
              };
              fetch("../cd", options).then((res) => {});
            } else {
              const options = {
                method: "post",
                headers: {
                  Accept: "application/json, text/plain, */*",
                  "Content-Type": "application/json",
                },
                body: JSON.stringify({
                  command: terminalInput.value,
                }),
              };
              fetch("../run-command", options).then((res) => {});
            }
            terminalBottom.scrollIntoView(false);
            terminalInput.value = "";
          }
        }
      },
      false
    );
    setLoaded(true);
  }, []);
  return (
    <div
      style={{
        width: "100%",
        height: "100vh",
        backgroundColor: colors.colorDark1,
      }}
    >
      <PoolConfigurator />
      {loaded ? (
        <Websocket
          url="ws://localhost:3001/"
          onMessage={(data) => {
            let packet = JSON.parse(data);
            if (packet.type === "terminal") {
              terminalLines.push({
                type: "system",
                content: packet.content,
              });
              if (document.getElementById("terminalBottom") !== null) {
                document.getElementById("terminalBottom").scrollIntoView(false);
              }
              terminalPrompt = packet.path;
              if (!terminalIsBusy) {
                terminalIsBusy = true;
              }
              forceUpdateApp();
            } else if (packet.type === "terminal-close") {
              terminalIsBusy = false;
              forceUpdateApp();
            } else if (packet.type === "files") {
              setFileTree(packet.tree);
            } else if (packet.type === 'resources') {
              setCpu(packet.cpu.percent);
              setMem(packet.memory.used * 100 / packet.memory.total);
            }
          }}
        />
      ) : null}
      <MainAppBar onDrawerBtnClicked={() => {setDrawerOpen(true);}}/>
      <MainDrawer drawerOpen={drawerOpen} setDrawerOpen={setDrawerOpen} />
      <div
        style={{
          width: "100%",
          height: "calc(100% - 112px)",
          display: "flex",
        }}
      >
        <div style={{ width: 350, height: "100%" }}>
          <Paper
            style={{
              marginTop: 16,
              marginLeft: 8,
              width: "100%",
              height: "50%",
              backgroundColor: colors.colorDark2,
              padding: 16,
              borderRadius: 16,
            }}
          >
            <CompView itemId={currentItemId} />
          </Paper>
          <Paper
            style={{
              marginTop: 16,
              marginLeft: 8,
              width: "100%",
              height: "50%",
              backgroundColor: colors.colorDark2,
              padding: 16,
              borderRadius: 16,
            }}
          >
            <FileView fileTree={fileTree} />
          </Paper>
        </div>
        <div
          style={{
            width: "calc(100% - 350px - 16px - 16px)",
            height: "100%",
            marginLeft: 4,
            marginTop: 16,
          }}
        >
          <div
            style={{
              width: "100%",
              height: "55vh",
              marginLeft: 16,
              display: "flex",
            }}
          >
            <Paper
              style={{
                width: "calc(100% - 350px)",
                height: "100%",
                backgroundColor: colors.colorDark2,
                padding: 16,
                borderRadius: 16,
              }}
            >
              {pool === undefined
                ? null
                : Object.keys(pool).map((path) => {
                    return (
                      <div
                        style={{
                          width: "100%",
                          height: "100%",
                          display: currentItemId === path ? "block" : "none",
                        }}
                      >
                        <CodeEditor itemId={path} />
                      </div>
                    );
                  })}
            </Paper>
            <Paper
              style={{
                width: 350,
                height: "100%",
                backgroundColor: colors.colorDark2,
                padding: 16,
                borderRadius: 16,
                marginLeft: 16,
              }}
            >
              <ObjectStore />
            </Paper>
          </div>
          <div
            style={{
              width: "100%",
              height: "calc(45vh - 112px)",
              marginTop: 16,
              marginLeft: 16,
              display: "flex",
            }}
          >
            <Paper
              style={{
                width: "calc(100% - 450px)",
                height: "100%",
                backgroundColor: colors.colorDark2,
                padding: 16,
                borderRadius: 16,
              }}
            >
              <Terminal
                onTerminalFocused={v => {
                  terminalInputFocused = v;
                }}
                terminalPrompt={terminalPrompt}
                terminalLines={terminalLines}
                terminalIsBusy={terminalIsBusy}
                terminalInputFocused={terminalInputFocused}
              />
            </Paper>
            <Paper
              style={{
                marginLeft: 12,
                width: 450,
                height: "100%",
                backgroundColor: colors.colorDark2,
                padding: 16,
                borderRadius: 16,
              }}
            >
              <ResMonitor cpu={cpu} mem={mem} />
            </Paper>
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
