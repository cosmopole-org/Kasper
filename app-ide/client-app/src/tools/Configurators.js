import { colors } from "../App";
import { getPoolItem, pool } from "../memory";

if(Array.prototype.equals)
    console.warn("Overriding existing Array.prototype.equals. Possible causes: New API defines the method, there's a framework conflict or you've got double inclusions in your code.");
// attach the .equals method to Array's prototype to call it on any array
Array.prototype.equals = function (array) {
    // if the other array is a falsy value, return
    if (!array)
        return false;

    // compare lengths - can save a lot of time 
    if (this.length != array.length)
        return false;

    for (var i = 0, l=this.length; i < l; i++) {
        // Check if we have nested arrays
        if (this[i] instanceof Array && array[i] instanceof Array) {
            // recurse into the nested arrays
            if (!this[i].equals(array[i]))
                return false;       
        }           
        else if (this[i] != array[i]) { 
            // Warning - two different object instances will never be equal: {x:20} != {x:20}
            return false;   
        }           
    }       
    return true;
}
// Hide method from for-in loops
Object.defineProperty(Array.prototype, "equals", {enumerable: false});

let keywords = [
  "use",
  "command",
  "times",
  "instance",
  "of",
  "define",
  "return",
  "prop",
  "on",
  "created",
  "class",
  "function",
  "with",
  "params",
  "for",
  "until",
  "if",
  "else",
  "then",
  "do",
  "true",
  "false",
  "not",
  "by",
  "step"
];
let monarchTokenProvider = {
  keywords: keywords,

  typeKeywords: [],

  operators: [
    "=",
    ">",
    "<",
    "!",
    "~",
    "?",
    ":",
    "==",
    "<=",
    ">=",
    "!=",
    "++",
    "--",
    "+",
    "-",
    "*",
    "/",
    "^",
    "<<",
    ">>",
    ">>>",
    "+=",
    "-=",
    "*=",
    "/=",
    "&=",
    "|=",
    "^=",
    "mod=",
    "<<=",
    ">>=",
    ">>>=",
  ],

  // we include these common regular expressions
  symbols: /[=><!~?:&|+\-*\/\^%]+/,

  // C# style strings
  escapes:
    /\\(?:[abfnrtv\\"']|x[0-9A-Fa-f]{1,4}|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8})/,

  // The main tokenizer for our languages
  tokenizer: {
    root: [
      // strings
      [/"/, { token: "string.quote", bracket: "@open", next: "@string" }],

      [/define[\s]+command[\s]+{.*/, { token: "command" }],

      // identifiers and keywords
      [
        /[a-z_$][\w$]*/,
        {
          cases: {
            "@typeKeywords": "keyword",
            "@keywords": "keyword",
            "@default": "identifier",
          },
        },
      ],
      [/[A-Z][\w\$]*/, "type.identifier"], // to show class names nicely

      // whitespace
      { include: "@whitespace" },

      // delimiters and operators
      [/[{}()\[\]]/, "brackets"],
      [/[<>](?!@symbols)/, "brackets"],
      [/@symbols/, { cases: { "@operators": "operator", "@default": "" } }],

      // @ annotations.
      // As an example, we emit a debugging log message on these tokens.
      // Note: message are supressed during the first load -- change some lines to see them.
      [/@\s*[a-zA-Z_\$][\w\$]*/, { token: "annotation" }],

      // numbers
      [/\d*\.\d+([eE][\-+]?\d+)?/, "number.float"],
      [/0[xX][0-9a-fA-F]+/, "number.hex"],
      [/\d+/, "number"],

      // delimiter: after number because of .\d floats
      [/[;,.]/, "delimiter"],
    ],

    comment2: [
      [/[^\/*]+/, "comment2"],
      [/\/\*/, "comment2", "@push"],
      ["\\*/", "comment2", "@pop"],
      [/[\/*]/, "comment2"],
    ],

    string: [
      [/[^\\"]+/, "string"],
      [/@escapes/, "string.escape"],
      [/\\./, "string.escape.invalid"],
      [/"/, { token: "string.quote", bracket: "@close", next: "@pop" }],
    ],

    whitespace: [
      [/[ \t\r\n]+/, "white"],
      [/\/\*/, "comment2", "@comment2"],
      [/\/\/.*$/, "comment2"],
    ],
  },
};

export const initiateLanguage = (monaco) => {
  monaco.languages.register({ id: "Elpis" });

  monaco.languages.setMonarchTokensProvider("Elpis", monarchTokenProvider);

  monaco.languages.registerCompletionItemProvider("Elpis", {
    provideCompletionItems: () => {
      var suggestions = [
        {
          label: "ifelse",
          kind: monaco.languages.CompletionItemKind.Snippet,
          insertText: [
            "if ${1:condition} then do",
            "\t$0",
            "else do",
            "\t",
            "",
          ].join("\n"),
          insertTextRules:
            monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          documentation: "If-Else Statement",
        },
        {
          label: "ifelseif",
          kind: monaco.languages.CompletionItemKind.Snippet,
          insertText: [
            "if ${1:condition} then do",
            "\t$0",
            "else if ${2:condition} then do",
            "\t",
            "",
          ].join("\n"),
          insertTextRules:
            monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          documentation: "If-ElseIf Statement",
        },
        {
          label: "ifelseIfelse",
          kind: monaco.languages.CompletionItemKind.Snippet,
          insertText: [
            "if ${1:condition} then do",
            "\t$0",
            "else if ${2:condition} then do",
            "\t",
            "else do",
            "\t",
            "",
          ].join("\n"),
          insertTextRules:
            monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          documentation: "If-Else Statement",
        },
      ];
      return { suggestions: suggestions };
    },
  });
};

export const initiateTheme = (monaco) => {
  monaco.editor.defineTheme("ElpisTheme", {
    base: "vs-dark",
    inherit: true,
    rules: [
      { token: "keyword", foreground: "FFA500", fontStyle: "bold" },
      { token: "operator", foreground: "FFA500", fontStyle: "bold" },
      { token: "brackets", foreground: "00FFFF", fontStyle: "bold" },
      { token: "command", foreground: "0099FF", fontStyle: "bold" },
      { token: "comment2", foreground: "00FF99", fontStyle: "bold" },
      { background: colors.colorDark2 },
    ],
    colors: {
      "editor.background": colors.colorDark2,
      "editor.foreground": "#d9d7ce",
      "editorIndentGuide.background": "#393b41",
      "editorIndentGuide.activeBackground": "#494b51",
    },
  });
};

export const findKeywords = (itemId) => {
  findKeywordsByCode(pool[itemId].editor.getValue(), pool[itemId].monaco);
}

export const findKeywordsByCode = (code, monaco) => {
  let occ = code.match(/(\bdefine[\s]+command[\s]+{[\s]+[A-Za-z0-9 ]+[\s]+})/g);
  if (occ !== null) {
    let filtered = [];
    occ.forEach((item) => {
      item = item.trim().substring("define".length);
      item = item.trim().substring("command".length);
      item = item.trim().substring(1, item.length - 1);
      let items = item.trim().split(" ");
      items.forEach((miniItem) => {
        if (miniItem.trim().length > 0) filtered.push(miniItem.trim());
      });
    });
    if (filtered.length > 0) {
      let newKeywords = [];
      keywords.forEach((keyword) => {
        newKeywords.push(keyword);
      });
      let newFound = false;
      filtered.forEach((item) => {
        if (!newKeywords.includes(item)) {
          newFound = true;
          newKeywords.push(item);
        }
      });
      if (!newKeywords.equals(monarchTokenProvider.keywords)) {
        monarchTokenProvider.keywords = newKeywords;
        monaco.languages.setMonarchTokensProvider(
          "Elpis",
          monarchTokenProvider
        );
      }
    }
  }
};

export const buildClasses = (code) => {
  let string = false;
  let listOfClasses = [];
  for (let counter = 0; counter < code.length; counter++) {
    if (code.charAt(counter) === '\\"') {
      continue;
    } else if (code.charAt(counter) === '"') {
      string = !string;
      continue;
    }
    if (!string && code.substring(counter).startsWith("define")) {
      let lineStart = counter;
      let spaceCount = 0;
      while (lineStart > 0 && code.charAt(lineStart - 1) === " ") {
        spaceCount++;
        lineStart--;
      }
      counter += "define ".length;
      let partly = code.substring(counter);
      let type = "";
      if (partly.startsWith("class")) {
        type = "class";
        counter += "class ".length;
        partly = code.substring(counter);
      } else if (partly.startsWith("function")) {
        type = "function";
        counter += "function ".length;
        partly = code.substring(counter);
      } else {
        continue;
      }
      let name = "";
      try {
        name = partly.match(/^[\w\d]+/gs)[0];
      } catch (ex) {}
      listOfClasses.push({ spaceCount: spaceCount, name: name, type: type });
    }
  }
  let components = [];
  let prevItem = null;
  let pointingItem = null;
  listOfClasses.forEach((item) => {
    item.children = [];
    item.parent = null;
    if (prevItem === null) {
      pointingItem = item;
      components.push(item);
    } else {
      if (item.spaceCount > pointingItem.spaceCount) {
        pointingItem.children.push(item);
        item.parent = pointingItem;
      } else if (item.spaceCount === pointingItem.spaceCount) {
        pointingItem = pointingItem.parent;
        if (pointingItem === null) {
          pointingItem = item;
          components.push(item);
        } else {
          pointingItem.children.push(item);
          item.parent = pointingItem;
        }
      } else if (item.spaceCount < pointingItem.spaceCount) {
        while (pointingItem.spaceCount >= item.spaceCount) {
          pointingItem = pointingItem.parent;
        }
        if (pointingItem === null) {
          pointingItem = item;
          components.push(item);
        } else {
          pointingItem.children.push(item);
          item.parent = pointingItem;
        }
      }
    }
    prevItem = item;
  });
  let rootObj = { name: "components", children: components };
  return rootObj;
};
