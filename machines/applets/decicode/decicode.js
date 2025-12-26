
let uiInstructions = `
    to generate a component assume the code should a javascript and json code and only based on these elements:
        (type: text (showing a text in ui), properties: [
            content ( which is string that is the text being shown ),
            textColor ( which is a hex color code specifying text color ),
            textSize ( which is an integer showing font size )
        ]),
        (type: image (showing an image in ui), properties: [
            url ( which is string url of image link ),
        ]),
        (type: button (an interactive button element in ui which is clickable), properties: [
            label ( which is string label being shown on the button ),
            textColor ( which is a hex color code specifying text color ),
            textSize ( which is an integer showing font size,
        ]),
        (type: container ( a container element which has same use case as div in html ), properties: [
            width ( which is integer width of container ),
            height ( which is integer height of container ),
            padding: ( a json object that contains for integer fields "left", "top", "right", "bottom" as padding in 4 direction )
            bgcolor: ( a string showing hex color code of container background color )
            child ( which can be any of the other elements in the current instructions, a container can have a single child which is a json object ),
        ]),
        (type: array ( it is similar to container element but can have multiple children and the field name is "items" also it can be vertical or horizontal ), properties: [
            orientation ( it is a string that can be 'horizontal' or 'vertical' showing direction of showing items in ui ),
            items ( which can be an array of any of the other elements or mix of them ),
        ]),
        remember type property and other properties are next to each other in a json object. produce this json object in function called "comp", the json object must be nested in anothe json object and assigned to field named 'root', return the json object from that function and put this code after that using comp function:
        """"
        if (!started) {
            initApp(comp());
        } else {
            render(comp());
        }""""
        the template is like this:
        """"
        function comp() {
            return {
                root: ""json object""
            };
        }
        if (!started) {
            initApp(comp());
        } else {
            render(comp());
        }""""
        also remember each time you modify values and wanna update and rerender the ui, you should call "render(comp())".
`;

function comp() {
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
                                    treeData: cache["docsTree"],
                                    itemBuilder: (key, data, level) => {
                                        let doc = JSON.parse(data);
                                        return joinUI({
                                            type: 'popupMenu',
                                            button: (() => {
                                                if (doc.isDir) {
                                                    return joinUI(
                                                        {
                                                            type: 'row',
                                                            children: [
                                                                {
                                                                    type: 'text',
                                                                    data: doc.title
                                                                },
                                                                {
                                                                    type: 'iconButton',
                                                                    icon: {
                                                                        "type": "icon",
                                                                        "icon": "add"
                                                                    },
                                                                    onPressed: () => {
                                                                        if (!cache["isMenuOpen"]) {
                                                                            cache["isMenuOpen"] = true;
                                                                            openMenu();
                                                                        }
                                                                    }
                                                                }
                                                            ]
                                                        }
                                                    );
                                                } else {
                                                    return joinUI(
                                                        {
                                                            type: 'row',
                                                            children: [
                                                                {
                                                                    type: 'elevatedButton',
                                                                    child: {
                                                                        type: 'text',
                                                                        data: doc.title
                                                                    },
                                                                    onPressed: () => {
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
                                                                                cache["sandobxRerenderFlag"] = Math.random().toString();
                                                                                render(comp());
                                                                            }
                                                                        });
                                                                    }
                                                                },
                                                                {
                                                                    type: 'iconButton',
                                                                    icon: {
                                                                        "type": "icon",
                                                                        "icon": "add"
                                                                    },
                                                                    onPressed: () => {
                                                                        if (!cache["isMenuOpen"]) {
                                                                            cache["isMenuOpen"] = true;
                                                                            openMenu();
                                                                        }
                                                                    }
                                                                }
                                                            ]
                                                        }
                                                    );
                                                }
                                            })(),
                                            items: doc.isDir ? [
                                                {
                                                    "value": "option1",
                                                    "label": "new file",
                                                },
                                                {
                                                    "value": "option1",
                                                    "label": "new folder",
                                                },
                                                {
                                                    "value": "option1",
                                                    "label": "delete",
                                                },
                                            ] : [
                                                {
                                                    "value": "option1",
                                                    "label": "delete",
                                                },
                                            ],
                                            onItemPress: (index) => {
                                                if (doc.isDir) {
                                                    if (index === 0) {
                                                        openCustomDialog(
                                                            "Create new file",
                                                            joinUI({
                                                                type: 'array',
                                                                orientation: 'vertical',
                                                                items: [
                                                                    {
                                                                        type: 'text',
                                                                        content: 'enter file name:'
                                                                    },
                                                                    {
                                                                        type: 'input',
                                                                        key: 'createFileNameInput',
                                                                        hint: 'type file name',
                                                                        onChange: (text) => {
                                                                            cache["creatingFileNameInput"] = text;
                                                                        },
                                                                    }
                                                                ]
                                                            }),
                                                            (closeDialog) => [
                                                                joinUI({
                                                                    type: 'button',
                                                                    label: 'cancel',
                                                                    onPress: () => {
                                                                        cache["creatingFileNameInput"] = "";
                                                                        closeDialog();
                                                                    }
                                                                }),
                                                                joinUI({
                                                                    type: 'button',
                                                                    label: 'create',
                                                                    onPress: () => {
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
                                                            ]
                                                        )
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
                                                    key: 'buildMachineInput',
                                                    hint: 'type machine id',
                                                    onChange: (text) => {
                                                        cache["builingMachineInput"] = text;
                                                    },
                                                }
                                            ]
                                        }),
                                        (closeDialog) => [
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
                                                        base64Encode(cache["currentCode"], (b64Encoded) => {
                                                            console.log(b64Encoded);
                                                            sendRequest("/storage/uploadUserEntity", { entityId: 'widget', data: b64Encoded, machineId: cache["builingMachineInput"] }, '', () => {
                                                                cache["builingMachineInput"] = "";
                                                                closeDialog();
                                                            });
                                                        });
                                                    }
                                                }
                                            })
                                        ]
                                    )
                                }
                            },
                        ]
                    }
                },
                {
                    type: 'code',
                    key: "mainCode",
                    minLines: 35,
                    width: meta.width - 250 - 350,
                    height: meta.height,
                    code: cache["currentCode"] ?? "",
                    onChange: (text) => {
                        console.log(text);
                        cache["currentCode"] = text;
                        if (cache["updaterActive"]) {
                            if (cache["codeUpdater"]) {
                                clearTimeout(cache["codeUpdater"]);
                            }
                            cache["codeUpdater"] = setTimeout(() => {
                                cache["codeUpdater"] = undefined;
                                console.log("updating... ");
                                ask(cache["workspaceId"], { type: 'updateCodeFile', filePath: cache["currentPath"], code: cache["currentCode"] }, () => { });
                            }, 1000);
                        }
                    }
                },
                {
                    type: "container",
                    width: 16,
                    height: meta.height
                },
                {
                    type: 'column',
                    items: [
                        {
                            type: 'container',
                            height: 250,
                            width: 350,
                            child: {
                                type: 'row',
                                items: [
                                    {
                                        type: "container",
                                        width: 48,
                                        height: meta.height
                                    },
                                    {
                                        type: 'sandbox',
                                        key: 'preview',
                                        rerenderFlag: cache["sandobxRerenderFlag"],
                                        width: 250,
                                        height: 250,
                                        entityId: meta.userId + "_" + (cache["currentPath"].replace("/", "_")),
                                        machineId: "61@global"
                                    },
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
                                    height: meta.height - 320,
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
    cache["docs"] = [];
    cache["docsTree"] = { children: [], key: '0', data: JSON.stringify({ path: "", title: "loading...", id: "0" }), title: "src", path: "", id: "0", };
    listen("codeUpdated", (packet) => {
        if (packet.updatedBy === meta.userId) return;
        let filePath = packet.filePath;
        if (cache["currentPath"] === filePath) {
            cache["currentPath"] = '';
            render(comp());
            setTimeout(() => {
                cache["sandobxRerenderFlag"] = Math.random().toString();
                cache["currentPath"] = filePath;
                cache["currentCode"] = packet.code;
                render(comp());
            }, 2500);
        }
    });
    ask(meta.pointId, { type: 'initWorkspace' }, (workspace) => {
        cache["workspaceId"] = workspace.Id;
        initApp(comp());
        ask(cache["workspaceId"], { type: 'files.read' }, (docs) => {
            cache["docs"] = docs;
            buildDocsTree();
            render(comp());
        });
    });
} else {
    render(comp());
}
