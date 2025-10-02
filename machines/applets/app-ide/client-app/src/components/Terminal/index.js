import React, { useEffect, useRef } from "react";
import { colors, forceUpdateApp } from "../../App";

export default function Terminal(props) {
  let terminal = useRef(null);
  let terminalInput = useRef(null);
  return (
    <div
      onClick={() => terminalInput.focus()}
      style={{
        width: "100%",
        display: "flex",
        height: "100%",
        overflowY: "auto",
        cursor: "text",
      }}
    >
      <ul
        id={"terminal"}
        ref={(el) => {
          terminal = el;
        }}
        style={{
          listStyleType: "none",
          color: "white",
        }}
      >
        {props.terminalLines.map((line) => {
          if (line.type === "system") {
            return (
              <li style={{ color: "white" }}>
                <pre>{line.content}</pre>
              </li>
            );
          } else {
            return (
              <li style={{ color: "#ffd700" }}>
                <pre>{line.content}</pre>
              </li>
            );
          }
        })}
        <div
          style={{
            display: "flex",
            color: "white",
          }}
        >
          {props.terminalIsBusy ? null : (
            <div style={{ color: "#ffd700" }}>{props.terminalPrompt}</div>
          )}
          <input
            id={"terminalInput"}
            type="text"
            style={{
              marginBottom: 32,
              background: "transparent",
              border: "none",
              color: "white",
              outline: "none",
              marginLeft: 16,
            }}
            onFocus={() => {
              props.onTerminalFocused(true);
              forceUpdateApp();
            }}
            onBlur={() => {
              props.onTerminalFocused(false);
              forceUpdateApp();
            }}
            ref={(el) => {
              terminalInput = el;
            }}
          />
        </div>
        <div
          id={"terminalBottom"}
          style={{
            width: "100%",
            height: 16,
          }}
        />
      </ul>
    </div>
  );
}
