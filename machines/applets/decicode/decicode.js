
function comp() {
    return {
        root: {
            type: "container",
            height: meta.height,
            width: meta.width,
            child: {
                type: 'array',
                orientation: 'horizontal',
                items: [
                    {
                        type: 'container',
                        width: 250,
                        padding: {
                            top: 16,
                            left: 16
                        },
                        height: meta.height,
                        child: {
                            type: 'array',
                            orientation: 'vertical',
                            items: [
                                {
                                    type: 'array',
                                    orientation: 'horizontal',
                                    items: [
                                        {
                                            type: 'button',
                                            label: '+ file',
                                            onPress: () => {
                                                ask(cache["workspaceId"], { type: 'files.create', isDir: false, docTitle: 'hello.js', docPath: "0" }, (docs) => {
                                                    cache["docs"] = docs;
                                                    buildDocsTree();
                                                    updateApp(comp());
                                                });
                                            }
                                        },
                                        {
                                            type: 'container',
                                            width: 8,
                                            height: 16
                                        },
                                        {
                                            type: 'button',
                                            label: '+ folder',
                                            onPress: () => {
                                                ask(cache["workspaceId"], { type: 'files.create', isDir: true, docTitle: 'hello.js', docPath: "0" }, (docs) => {
                                                    cache["docs"] = docs;
                                                    buildDocsTree();
                                                    updateApp(comp());
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
                                        type: 'treeView',
                                        treeData: cache["docsTree"],
                                        itemBuilder: (key, data, level) => {
                                            log("hi " + data);
                                            let doc = JSON.parse(data);
                                            return scanComp({
                                                type: 'popupMenu',
                                                menuButton: (openMenu, closeMenu) => {
                                                    if (doc.isDir) {
                                                        return scanComp(
                                                            {
                                                                type: 'array',
                                                                orientation: 'horizontal',
                                                                items: [
                                                                    {
                                                                        type: 'text',
                                                                        content: doc.title
                                                                    },
                                                                    {
                                                                        type: 'iconButton',
                                                                        icon: "more_vert",
                                                                        glass: false,
                                                                        isTransparent: true,
                                                                        onPress: () => {
                                                                            log("keyhan 1");
                                                                            if (cache["isMenuOpen"]) {
                                                                                cache["isMenuOpen"] = false;
                                                                                closeMenu();
                                                                            } else {
                                                                                cache["isMenuOpen"] = true;
                                                                                openMenu();
                                                                            }
                                                                        }
                                                                    }
                                                                ]
                                                            }
                                                        );
                                                    } else {
                                                        return scanComp(
                                                            {
                                                                type: 'array',
                                                                orientation: 'horizontal',
                                                                items: [
                                                                    {
                                                                        type: 'button',
                                                                        label: doc.title,
                                                                        onPress: () => {
                                                                            cache["currentCode"] = "let a = 2;";
                                                                            updateApp(comp());
                                                                        }
                                                                    },
                                                                    {
                                                                        type: 'iconButton',
                                                                        icon: "more_vert",
                                                                        glass: false,
                                                                        isTransparent: true,
                                                                        onPress: () => {
                                                                            log("keyhan 1");
                                                                            if (cache["isMenuOpen"]) {
                                                                                cache["isMenuOpen"] = false;
                                                                                closeMenu();
                                                                            } else {
                                                                                cache["isMenuOpen"] = true;
                                                                                openMenu();
                                                                            }
                                                                        }
                                                                    }
                                                                ]
                                                            }
                                                        );
                                                    }
                                                },
                                                itemCount: doc.isDir ? 3 : 1,
                                                itemBuilder: (index) => {
                                                    if (doc.isDir) {
                                                        if (index == 0) {
                                                            return {
                                                                type: 'text',
                                                                content: 'new file',
                                                            }
                                                        } else if (index == 1) {
                                                            return {
                                                                type: 'text',
                                                                content: 'new folder',
                                                            }
                                                        } else if (index == 2) {
                                                            return {
                                                                type: 'text',
                                                                content: 'delete',
                                                            }
                                                        }
                                                    } else {
                                                        if (index == 0) {
                                                            return {
                                                                type: 'text',
                                                                content: 'delete',
                                                            }
                                                        }
                                                    }
                                                },
                                                onItemPress: (index) => {
                                                    if (doc.isDir) {
                                                        if (index == 0) {
                                                            openCustomDialog(
                                                                "Create new file",
                                                                scanComp({
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
                                                                    scanComp({
                                                                        type: 'button',
                                                                        label: 'cancel',
                                                                        onPress: () => {
                                                                            cache["creatingFileNameInput"] = "";
                                                                            closeDialog();
                                                                        }
                                                                    }),
                                                                    scanComp({
                                                                        type: 'button',
                                                                        label: 'create',
                                                                        onPress: () => {
                                                                            if (cache["creatingFileNameInput"] && cache["creatingFileNameInput"].length > 0) {
                                                                                ask(cache["workspaceId"], { type: 'files.create', isDir: false, docTitle: cache["creatingFileNameInput"], docPath: doc.path + (doc.path.length > 0 ? "/" : "") + doc.id }, (docs) => {
                                                                                    cache["creatingFileNameInput"] = "";
                                                                                    cache["docs"] = docs;
                                                                                    closeDialog();
                                                                                    buildDocsTree();
                                                                                    updateApp(comp());
                                                                                });
                                                                            }
                                                                        }
                                                                    })
                                                                ]
                                                            )
                                                        } else if (index == 1) {
                                                            ask(cache["workspaceId"], { type: 'files.create', isDir: true, docTitle: 'hello.js', docPath: doc.path + (doc.path.length > 0 ? "/" : "") + doc.id }, (docs) => {
                                                                cache["docs"] = docs;
                                                                buildDocsTree();
                                                                updateApp(comp());
                                                            });
                                                        } else if (index == 2) {
                                                            ask(cache["workspaceId"], { type: 'files.delete', docId: doc.id }, (docs) => {
                                                                cache["docs"] = docs;
                                                                buildDocsTree();
                                                                updateApp(comp());
                                                            });
                                                        }
                                                    } else {
                                                        if (index == 0) {
                                                            ask(cache["workspaceId"], { type: 'files.delete', docId: doc.id }, (docs) => {
                                                                cache["docs"] = docs;
                                                                buildDocsTree();
                                                                updateApp(comp());
                                                            });
                                                        }
                                                    }
                                                }
                                            });
                                        },
                                        onItemTap: (key) => {
                                            log(key + " tapped !");
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    {
                        type: 'codeEditor',
                        key: "mainCode",
                        minLines: 35,
                        width: meta.width - 250 - 350,
                        height: meta.height,
                        code: cache["currentCode"] ?? "",
                        onChange: (text) => {
                            cache["currentCode"] = text;
                        }
                    },
                    {
                        type: "container",
                        width: 16,
                        height: meta.height
                    },
                    {
                        type: 'array',
                        orientation: 'vertical',
                        items: [
                            {
                                type: 'container',
                                height: 250,
                                width: 350,
                                child: {
                                    type: 'array',
                                    orientation: 'horizontal',
                                    items: [
                                        {
                                            type: "container",
                                            width: 48,
                                            height: meta.height
                                        },
                                        {
                                            type: 'container',
                                            width: 250,
                                            height: 250,
                                            child: {
                                                type: "container",
                                                width: 250,
                                                height: 250,
                                                child: {
                                                    type: "glassContainer",
                                                    borderRadius: 250 / 2,
                                                    child: {
                                                        type: "container",
                                                        height: 250,
                                                        width: 250,
                                                        shape: "circle",
                                                        borderColor: "#999999",
                                                        child: {
                                                            type: 'freelayout',
                                                            items: [
                                                                {
                                                                    type: "container",
                                                                    left: 0,
                                                                    top: 0,
                                                                    width: 250,
                                                                    height: 250,
                                                                    child: {
                                                                        type: 'freelayout',
                                                                        items: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11].map((index) => (
                                                                            {
                                                                                type: "container",
                                                                                left: 0,
                                                                                top: 0,
                                                                                width: 250,
                                                                                height: 250,
                                                                                child: {
                                                                                    type: "container",
                                                                                    center: true,
                                                                                    transform: {
                                                                                        rotation: index * 30 - 90
                                                                                    },
                                                                                    child: {
                                                                                        type: "container",
                                                                                        width: 250,
                                                                                        height: 24,
                                                                                        child: {
                                                                                            type: 'freelayout',
                                                                                            items: [
                                                                                                {
                                                                                                    type: "container",
                                                                                                    left: 250 - 32,
                                                                                                    child: {
                                                                                                        type: "text",
                                                                                                        content: (index == 0 ? 12 : index).toString(),
                                                                                                        transform: {
                                                                                                            rotation: -(index * 30 - 90)
                                                                                                        },
                                                                                                    }
                                                                                                }
                                                                                            ]
                                                                                        }
                                                                                    }
                                                                                }
                                                                            }
                                                                        ))
                                                                    }
                                                                },
                                                                {
                                                                    type: "container",
                                                                    left: 0,
                                                                    top: 0,
                                                                    width: 250,
                                                                    height: 250,
                                                                    child: {
                                                                        type: "container",
                                                                        center: true,
                                                                        transform: {
                                                                            rotation: 50
                                                                        },
                                                                        child: {
                                                                            type: "container",
                                                                            width: 250 * 3 / 5,
                                                                            height: 12,
                                                                            child: {
                                                                                type: 'freelayout',
                                                                                items: [
                                                                                    {
                                                                                        type: "container",
                                                                                        left: (250 * 3 / 5) / 2,
                                                                                        width: (250 * 3 / 5) / 2,
                                                                                        height: 12,
                                                                                        bgcolor: meta.primaryColor1,
                                                                                        borderRadius: 6,
                                                                                    }
                                                                                ]
                                                                            }
                                                                        }
                                                                    }
                                                                },
                                                                {
                                                                    type: "container",
                                                                    left: 0,
                                                                    top: 0,
                                                                    width: 250,
                                                                    height: 250,
                                                                    child: {
                                                                        type: "container",
                                                                        center: true,
                                                                        transform: {
                                                                            rotation: 25
                                                                        },
                                                                        child: {
                                                                            type: "container",
                                                                            width: 250 * 3.5 / 5,
                                                                            height: 8,
                                                                            child: {
                                                                                type: 'freelayout',
                                                                                items: [
                                                                                    {
                                                                                        type: "container",
                                                                                        left: (250 * 3.5 / 5) / 2,
                                                                                        width: (250 * 3.5 / 5) / 2,
                                                                                        height: 8,
                                                                                        bgcolor: meta.primaryColor2,
                                                                                        borderRadius: 4,
                                                                                    }
                                                                                ]
                                                                            }
                                                                        }
                                                                    }
                                                                },
                                                                {
                                                                    type: "container",
                                                                    left: 0,
                                                                    top: 0,
                                                                    width: 250,
                                                                    height: 250,
                                                                    child: {
                                                                        type: "container",
                                                                        center: true,
                                                                        transform: {
                                                                            rotation: 15
                                                                        },
                                                                        child: {
                                                                            type: "container",
                                                                            width: 250 * 4 / 5,
                                                                            height: 4,
                                                                            child: {
                                                                                type: 'freelayout',
                                                                                items: [
                                                                                    {
                                                                                        type: "container",
                                                                                        left: (250 * 4 / 5) / 2,
                                                                                        width: (250 * 4 / 5) / 2,
                                                                                        height: 4,
                                                                                        bgcolor: meta.primaryColor3,
                                                                                        borderRadius: 2,
                                                                                    }
                                                                                ]
                                                                            }
                                                                        }
                                                                    }
                                                                }
                                                            ]
                                                        }
                                                    }
                                                }
                                            }
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
                                type: 'glassContainer',
                                borderRadius: 16,
                                child: {
                                    type: 'container',
                                    width: 350,
                                    height: meta.height - 320,
                                    borderRadius: 16,
                                    padding: {
                                        left: 16,
                                        top: 16,
                                        right: 16,
                                        bottom: 16
                                    },
                                    child: {
                                        type: 'chat',
                                        pointId: cache["workspaceId"],
                                        onlyLLM: true
                                    }
                                }
                            }
                        ]
                    }
                ]
            }
        }
    };
}
function buildDocsTree() {
    log("oho");
    let root = { path: '', title: 'src', id: '0', children: {}, key: '0', data: JSON.stringify({ path: '', title: 'src', id: '0', isDir: true }) };
    cache["docs"].forEach((doc, index) => {
        log("hooooooooooooooo " + doc.Id);
        let p = doc.Path.substring(Math.min("0/".length, doc.Path.length));
        let temp = root.children;
        if (p.length > 0) {
            let pathParts = p.split("/");
            let progressPath = temp.id;
            for (let i in pathParts) {
                let part = pathParts[i];
                if (!temp[part]) {
                    log("haha " + part);
                    temp[part] = { path: progressPath, title: '', id: part, key: part, children: {} };
                }
                temp = temp[part].children;
                progressPath += '/' + part;
            }
        }
        if (temp[doc.Id]) {
            log("hihi " + temp[doc.Id].Id);
            temp[doc.Id].title = doc.Title;
            temp[doc.Id].data = JSON.stringify({ path: temp[doc.Id].path, title: doc.Title, id: temp[doc.Id].id, isDir: doc.IsDir });
        } else {
            log("hoohoo " + doc.Id);
            temp[doc.Id] = { path: doc.Path, title: doc.Title, id: doc.Id, children: {}, key: doc.Id, data: JSON.stringify({ path: doc.Path, title: doc.Title, id: doc.Id, isDir: doc.IsDir }) };
        }
    });
    log("oho 1")
    scanForTransform(root);
    cache["docsTree"] = root;
    log(JSON.stringify(cache["docsTree"]))
}
function scanForTransform(doc) {
    doc.children = Object.values(doc.children);
    doc.children.forEach(child => {
        scanForTransform(child);
    });
}
if (!started) {
    cache["docs"] = [];
    cache["docsTree"] = { children: [], key: '0', data: JSON.stringify({ path: "", title: "loading...", id: "0" }), title: "src", path: "", id: "0", };
    cache["messages"] = [];
    listen("codeUpdated", (packet) => {
        log("hi");
        log(JSON.stringify(packet));
        let filePath = packet.filePath;
        let code = packet.code;
        log(code);
        cache["currentCode"] = code;
        updateApp(comp());
    });
    ask(meta.pointId, { type: 'initWorkspace' }, (workspace) => {
        cache["workspaceId"] = workspace.Id;
        initApp(comp());
        ask(cache["workspaceId"], { type: 'files.read' }, (docs) => {
            log(JSON.stringify(docs));
            cache["docs"] = docs;
            buildDocsTree();
            updateApp(comp());
        });
    });
} else {
    updateApp(comp());
}