
let uiInstructions = `
    to generate a component assume the code should a javascript and json code and only based on "flutter stac sdui framework" json code.
    also remember each time you modify values and wanna update and rerender the ui, you should call "render(comp())".
`;

function comp() {
    console.log("kasper", meta.userId + "_" + (cache["currentPath"].replace("/", "_")));
    return {
        type: "container",
        height: meta.height,
        width: meta.width,
        child: {
            type: 'row',
            children: [
                {
                    type: 'container',
                    width: 250,
                    padding: {
                        top: 16,
                        left: 16
                    },
                    height: meta.height,
                    child: {
                        type: 'column',
                        children: [
                            {
                                type: 'row',
                                children: [
                                    {
                                        type: 'elevatedButton',
                                        child: {
                                            type: "text",
                                            data: '+ file',
                                        },
                                        onPressed: () => {
                                            ask(cache["workspaceId"], { type: 'files.create', isDir: false, docTitle: 'hello.js', docPath: "0" }, (docs) => {
                                                cache["docs"] = docs;
                                                buildDocsTree();
                                                render(comp());
                                            });
                                        }
                                    },
                                    {
                                        type: 'container',
                                        width: 8,
                                        height: 16
                                    },
                                    {
                                        type: 'elevatedButton',
                                        child: {
                                            type: "text",
                                            data: '+ folder',
                                        },
                                        onPressed: () => {
                                            ask(cache["workspaceId"], { type: 'files.create', isDir: true, docTitle: 'hello.js', docPath: "0" }, (docs) => {
                                                cache["docs"] = docs;
                                                buildDocsTree();
                                                render(comp());
                                            });
                                        }
                                    },
                                ]
                            },
                            {
                                type: 'container',
                                width: 250,
                                height: 32
                            },
                            {
                                type: 'container',
                                width: 250,
                                height: meta.height - 150,
                                child: {
                                    type: 'treeview',
                                    appId: meta.appId,
                                    treeData: cache["docsTree"],
                                    itemBuilder: (key, data, level) => {
                                        let doc = JSON.parse(data);
                                        return joinUI({
                                            type: 'row',
                                            children: [
                                                doc.isDir ?
                                                    {
                                                        type: 'text',
                                                        data: doc.title
                                                    } :
                                                    {
                                                        type: 'elevatedButton',
                                                        child: {
                                                            type: 'text',
                                                            data: doc.title
                                                        },
                                                        onPressed: () => {
                                                            if (!cache["codeLock"]) {
                                                                cache["codeBackup"] = cache["currentCode"];
                                                                console.log("updating... ");
                                                                ask(cache["workspaceId"], { type: 'updateCodeFile', filePath: cache["currentPath"], code: cache["currentCode"] }, () => { });
                                                            }
                                                            cache["mainCodeKey"] = Math.random().toString().substring(2);
                                                            cache["codeLock"] = true;
                                                            cache["updaterActive"] = false;
                                                            cache["currentCode"] = '';
                                                            cache["currentPath"] = doc.path + "/" + doc.id;
                                                            render(comp());
                                                            ask(cache["workspaceId"], { type: 'setCurrentPath', path: doc.path + "/" + doc.id }, () => { });
                                                            getEntity(meta.userId + "_" + (doc.path.replace("/", "_") + "_" + doc.id), (data) => {
                                                                if (cache["updaterFlagReturn"]) {
                                                                    clearTimeout(cache["updaterFlagReturn"]);
                                                                    cache["updaterFlagReturn"] = undefined;
                                                                }
                                                                cache["updaterFlagReturn"] = setTimeout(() => {
                                                                    cache["updaterFlagReturn"] = undefined;
                                                                    cache["updaterActive"] = true;
                                                                }, 1000);
                                                                if (cache["currentPath"] === (doc.path + "/" + doc.id)) {
                                                                    cache["currentCode"] = data;
                                                                    cache["currentCodeDef"] = data;
                                                                    cache["sandobxRerenderFlag"] = Math.random().toString();
                                                                    render(comp());
                                                                }
                                                                cache["codeLock"] = false;
                                                            });
                                                        }
                                                    },
                                                {
                                                    type: 'popupMenu',
                                                    appId: meta.appId,
                                                    button: () => {
                                                        return joinUI(
                                                            {
                                                                type: 'iconButton',
                                                                icon: {
                                                                    "type": "icon",
                                                                    "icon": "add"
                                                                },
                                                            }
                                                        );
                                                    },
                                                    items: doc.isDir ? [
                                                        {
                                                            "type": "text",
                                                            "data": "new file",
                                                        },
                                                        {
                                                            "type": "text",
                                                            "data": "new folder",
                                                        },
                                                        {
                                                            "type": "text",
                                                            "data": "delete",
                                                        },
                                                    ] : [
                                                        {
                                                            "type": "text",
                                                            "data": "delete",
                                                        },
                                                    ],
                                                    onItemPress: (index) => {
                                                        if (doc.isDir) {
                                                            if (index === 0) {
                                                                openCustomDialog(
                                                                    "Create new file",
                                                                    joinUI({
                                                                        type: 'column',
                                                                        children: [
                                                                            {
                                                                                type: 'text',
                                                                                data: 'enter file name:'
                                                                            },
                                                                            {
                                                                                type: 'input',
                                                                                appId: meta.appId,
                                                                                key: 'createFileNameInput',
                                                                                label: 'type file name',
                                                                                onChange: (text) => {
                                                                                    cache["creatingFileNameInput"] = text;
                                                                                },
                                                                            }
                                                                        ]
                                                                    }),
                                                                    (closeDialog) => {
                                                                        return [
                                                                            joinUI({
                                                                                type: 'elevatedButton',
                                                                                child: {
                                                                                    type: 'text',
                                                                                    data: 'cancel',
                                                                                },
                                                                                onPressed: () => {
                                                                                    cache["creatingFileNameInput"] = "";
                                                                                    closeDialog();
                                                                                }
                                                                            }),
                                                                            joinUI({
                                                                                type: 'elevatedButton',
                                                                                child: {
                                                                                    type: 'text',
                                                                                    data: 'create',
                                                                                },
                                                                                onPressed: () => {
                                                                                    if (cache["creatingFileNameInput"] && cache["creatingFileNameInput"].length > 0) {
                                                                                        ask(cache["workspaceId"], { type: 'files.create', isDir: false, docTitle: cache["creatingFileNameInput"], docPath: doc.path + (doc.path.length > 0 ? "/" : "") + doc.id }, (docs) => {
                                                                                            cache["creatingFileNameInput"] = "";
                                                                                            cache["docs"] = docs;
                                                                                            closeDialog();
                                                                                            buildDocsTree();
                                                                                            render(comp());
                                                                                        });
                                                                                    }
                                                                                }
                                                                            })
                                                                        ];
                                                                    })
                                                            } else if (index === 1) {
                                                                ask(cache["workspaceId"], { type: 'files.create', isDir: true, docTitle: 'hello.js', docPath: doc.path + (doc.path.length > 0 ? "/" : "") + doc.id }, (docs) => {
                                                                    cache["docs"] = docs;
                                                                    buildDocsTree();
                                                                    render(comp());
                                                                });
                                                            } else if (index === 2) {
                                                                ask(cache["workspaceId"], { type: 'files.delete', docId: doc.id }, (docs) => {
                                                                    cache["docs"] = docs;
                                                                    buildDocsTree();
                                                                    render(comp());
                                                                });
                                                            }
                                                        } else {
                                                            if (index === 0) {
                                                                ask(cache["workspaceId"], { type: 'files.delete', docId: doc.id }, (docs) => {
                                                                    cache["docs"] = docs;
                                                                    buildDocsTree();
                                                                    render(comp());
                                                                });
                                                            }
                                                        }
                                                    }
                                                }
                                            ]
                                        })
                                    },
                                    onItemTap: (key) => {
                                        console.log(key + " tapped !");
                                    }
                                }
                            },
                            {
                                type: 'elevatedButton',
                                child: {
                                    type: 'text',
                                    data: 'build',
                                },
                                onPressed: () => {
                                    openCustomDialog(
                                        "build",
                                        joinUI({
                                            type: 'column',
                                            children: [
                                                {
                                                    type: 'text',
                                                    data: 'enter machine id:'
                                                },
                                                {
                                                    type: 'input',
                                                    appId: meta.appId,
                                                    key: 'buildMachineInput',
                                                    label: 'type machine id',
                                                    onChange: (text) => {
                                                        cache["builingMachineInput"] = text;
                                                    },
                                                }
                                            ]
                                        }),
                                        (closeDialog) => ([
                                            joinUI({
                                                type: 'elevatedButton',
                                                child: {
                                                    type: 'text',
                                                    data: 'cancel',
                                                },
                                                onPressed: () => {
                                                    cache["builingMachineInput"] = "";
                                                    closeDialog();
                                                }
                                            }),
                                            joinUI({
                                                type: 'elevatedButton',
                                                child: {
                                                    type: 'text',
                                                    data: 'build',
                                                },
                                                onPressed: () => {
                                                    if (cache["builingMachineInput"] && cache["builingMachineInput"].length > 0) {
                                                        sendRequest("/storage/uploadUserEntity", { entityId: 'widget', data: cache["currentCode"], machineId: cache["builingMachineInput"] }, (res) => {
                                                            console.log("responseForm", res);
                                                            cache["builingMachineInput"] = "";
                                                            closeDialog();
                                                        });
                                                    }
                                                }
                                            })
                                        ])
                                    )
                                }
                            },
                        ]
                    }
                },
                {
                    type: 'codeSnippet',
                    appId: meta.appId,
                    key: cache["mainCodeKey"],
                    minLines: 35,
                    width: meta.width - 250 - 350 - 50,
                    height: meta.height,
                    data: cache["currentCodeDef"] ?? "",
                    onChange: (text) => {
                        cache["currentCode"] = text;
                        setTimeout(() => {
                            if (cache["currentCode"] === "" && !cache["codeLock"]) {
                                cache["codeBackup"] = cache["currentCode"];
                                console.log("updating... ");
                                ask(cache["workspaceId"], { type: 'updateCodeFile', filePath: cache["currentPath"], code: cache["currentCode"] }, () => { });
                            }
                        });
                    }
                },
                {
                    type: "container",
                    width: 16,
                    height: meta.height
                },
                {
                    type: 'column',
                    children: [
                        {
                            type: 'container',
                            width: 350,
                            height: 32
                        },
                        {
                            type: 'container',
                            height: 250,
                            width: 350,
                            child: {
                                type: 'row',
                                children: [
                                    {
                                        type: "container",
                                        width: 48,
                                        height: meta.height
                                    },
                                    cache["currentPath"] !== "" ? {
                                        type: 'sandbox',
                                        appId: meta.appId,
                                        key: 'preview',
                                        updateFlag: cache["sandobxRerenderFlag"],
                                        width: 250,
                                        height: 250,
                                        entityId: meta.userId + "_" + (cache["currentPath"].replace("/", "_")),
                                        machineId: "61@global"
                                    } : { type: "container" },
                                    {
                                        type: "container",
                                        width: 32,
                                        height: meta.height
                                    },
                                ]
                            },
                        },
                        {
                            type: 'container',
                            width: 350,
                            height: 32
                        },
                        {
                            type: "clipRRect",
                            borderRadius: 16,
                            clipBehavior: "antiAlias",
                            child: {
                                type: 'glassContainer',
                                child: {
                                    type: 'container',
                                    width: 350,
                                    height: meta.height - 280,
                                    decoration: {
                                        borderRadius: 16,
                                    },
                                    padding: {
                                        left: 16,
                                        top: 16,
                                        right: 16,
                                        bottom: 16
                                    },
                                    child: {
                                        type: 'chat',
                                        appId: meta.appId,
                                        pointId: cache["workspaceId"],
                                        onlyLLM: true,
                                        instructions: '"""current editting file path is ' + cache["currentPath"] + ' \n' + uiInstructions + ' \n also look at this code and use it as the context of existing code in the file and do all your work and updates on it: \n' + cache["currentCode"] + '""" '
                                    }
                                }
                            }
                        }
                    ]
                }
            ]
        }
    };
}
function buildDocsTree() {
    let root = { path: '', title: 'src', id: '0', children: {}, key: '0', data: JSON.stringify({ path: '', title: 'src', id: '0', isDir: true }) };
    cache["docs"].forEach((doc, index) => {
        let p = doc.Path.substring(Math.min("0/".length, doc.Path.length));
        let temp = root.children;
        if (p.length > 0) {
            let pathParts = p.split("/");
            let progressPath = temp.id;
            for (let i in pathParts) {
                let part = pathParts[i];
                if (!temp[part]) {
                    temp[part] = { path: progressPath, title: '', id: part, key: part, children: {} };
                }
                temp = temp[part].children;
                progressPath += '/' + part;
            }
        }
        if (temp[doc.Id]) {
            temp[doc.Id].title = doc.Title;
            temp[doc.Id].data = JSON.stringify({ path: temp[doc.Id].path, title: doc.Title, id: temp[doc.Id].id, isDir: doc.IsDir });
        } else {
            temp[doc.Id] = { path: doc.Path, title: doc.Title, id: doc.Id, children: {}, key: doc.Id, data: JSON.stringify({ path: doc.Path, title: doc.Title, id: doc.Id, isDir: doc.IsDir }) };
        }
    });
    scanForTransform(root);
    cache["docsTree"] = root;
}
function scanForTransform(doc) {
    doc.children = Object.values(doc.children);
    doc.children.forEach(child => {
        scanForTransform(child);
    });
}
if (!started) {
    cache["sandobxRerenderFlag"] = Math.random().toString();
    cache["updaterActive"] = true;
    cache["currentPath"] = '';
    cache["currentCode"] = '';
    cache["currentCodeDef"] = '';
    cache["mainCodeKey"] = Math.random().toString().substring(2);
    cache["docs"] = [];
    cache["docsTree"] = { children: [], key: '0', data: JSON.stringify({ path: "", title: "loading...", id: "0" }), title: "src", path: "", id: "0", };
    listen("codeUpdated", (packet) => {
        let filePath = packet.filePath;
        if (cache["sandboxBackup"] !== packet.code && cache["currentPath"] === filePath) {
            cache["sandboxBackup"] = packet.code;
            cache["sandobxRerenderFlag"] = Math.random().toString();
            render(comp());
        }
        if (packet.updatedBy === meta.userId) return;
        if (cache["currentPath"] === filePath) {
            cache["currentPath"] = '';
            render(comp());
            setTimeout(() => {
                cache["sandobxRerenderFlag"] = Math.random().toString();
                cache["currentPath"] = filePath;
                cache["mainCodeKey"] = Math.random().toString().substring(2);
                cache["currentCode"] = packet.code;
                render(comp());
            });
        }
    });
    ask(meta.pointId, { type: 'initWorkspace' }, (workspace) => {
        cache["workspaceId"] = workspace.Id;
        render(comp());
        ask(cache["workspaceId"], { type: 'files.read' }, (docs) => {
            cache["docs"] = docs;
            buildDocsTree();
            render(comp());
        });
    });
    setInterval(() => {
        if (cache["codeBackup"] !== cache["currentCode"]) {
            if (cache["currentCode"] !== "" && !cache["codeLock"]) {
                cache["codeBackup"] = cache["currentCode"];
                console.log("updating... ");
                ask(cache["workspaceId"], { type: 'updateCodeFile', filePath: cache["currentPath"], code: cache["currentCode"] }, () => { });
            }
        }
    }, 1000);
} else {
    render(comp());
}
