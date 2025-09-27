import React, { useEffect, useRef } from "react";
import ReactDOM from "react-dom";
import Editor from "@monaco-editor/react";
import { findKeywordsByCode, initiateLanguage, initiateTheme } from "../../tools/Configurators";
import { updatePool } from "../../memory";
import { buildClasses, findKeywords } from "../../tools/Configurators";
import { forceUpdateApp } from "../../App";

export default function CodeEditor({ itemId }) {
  const monacoRef = useRef(null);
  const editorRef = useRef(null);
  const [value, setValue] = React.useState('');

  function handleEditorWillMount(monaco) {
    initiateLanguage(monaco);
    initiateTheme(monaco);
    updatePool(itemId, {monaco});
    monacoRef.current = monaco;
    const options = {
      method: "post",
      headers: {
        Accept: "application/json, text/plain, */*",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        path: itemId,
      }),
    };
    fetch("../get-file", options)
      .then((rawRes) => {
        return rawRes.blob();
      })
      .then((blb) => {
        const reader = new FileReader();
        reader.addEventListener("loadend", (e) => {
          const text = e.srcElement.result;
          setValue(text);
          setTimeout(() => {
            buildClasses(text);
            findKeywordsByCode(text, monaco);
            forceUpdateApp();
          });
        });
        reader.readAsText(blb);
      });
    updatePool(itemId, {monaco});
  }

  function handleEditorDidMount(editor, monaco) {
    editorRef.current = editor;
    editor.onDidType((text) => {
      if (text === "\n") {
        var line = editor.getPosition();
        var text = editor.getValue(line);
        var splitedText = text.split("\n");
        var l = splitedText[line.lineNumber - 2];
        if (
          l.trimRight().endsWith(" do") ||
          l.trimRight().endsWith(" containing") ||
          l.trim().startsWith("define class") ||
          l.trim().startsWith("define function")
        ) {
          var range = new monaco.Range(line.lineNumber, 1, line.lineNumber, 1);
          var id = { major: 1, minor: 1 };
          var text = "  ";
          var op = {
            identifier: id,
            range: range,
            text: text,
            forceMoveMarkers: true,
          };
          editor.executeEdits("my-source", [op]);
        }
      }
    });
    updatePool(itemId, { editor });
  }

  return (
    <>
      <Editor
        style={{ borderRadius: 16 }}
        width="100%"
        height="50vh"
        language="Elpis"
        value={value}
        beforeMount={handleEditorWillMount}
        onMount={handleEditorDidMount}
        theme={"ElpisTheme"}
        onChange={() => {
          if (editorRef.current !== null) {
            setValue(editorRef.current.getValue());
            buildClasses(editorRef.current.getValue());
            findKeywords(itemId);
            forceUpdateApp();
            const options = {
              method: 'post',
              headers: {
                'Accept': 'application/json, text/plain, */*',
                'Content-Type': 'application/json'
              },
              body: JSON.stringify({
                "path": itemId,
                "content": editorRef.current.getValue()
              })
            };
            fetch("../update-code", options);
          }
        }}
      />
    </>
  );
}
