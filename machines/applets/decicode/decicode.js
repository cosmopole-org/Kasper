
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
                                                ask(cache["workspaceId"], { type: 'files.create', isDir: false, docTitle: 'hello.js', docPath: 'src/logic/' }, (docs) => {
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
                                                ask(cache["workspaceId"], { type: 'files.create', isDir: true, docTitle: 'hello.js', docPath: 'src/logic/' }, (docs) => {
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
                                    height: 16
                                },
                                {
                                    type: 'array',
                                    orientation: 'horizontal',
                                    items: [
                                        {
                                            type: 'button',
                                            label: 'clear',
                                            onPress: () => {
                                                ask(cache["workspaceId"], { type: 'files.purge' }, (docs) => {
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
                                                ask(cache["workspaceId"], { type: 'files.create', isDir: true, docTitle: 'hello.js', docPath: 'src/logic/' }, (docs) => {
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
                                        treeData: {
                                            key: 'root',
                                            data: 'src',
                                            children: []
                                        },
                                        itemBuilder: (key, data, level) => {
                                            log("hi " + data);
                                            return {
                                                type: 'text',
                                                content: data,
                                            };
                                            // let doc = JSON.parse(data);
                                            // return {
                                            //     type: 'popupMenu',
                                            //     menuButton: {
                                            //         type: 'text',
                                            //         content: doc.title,
                                            //     },
                                            //     itemCount: 2,
                                            //     itemBuilder: (index) => {
                                            //         if (index == 0) {
                                            //             return {
                                            //                 type: 'text',
                                            //                 content: 'new file',
                                            //             }
                                            //         } else if (index == 1) {
                                            //             return {
                                            //                 type: 'text',
                                            //                 content: 'new folder',
                                            //             }
                                            //         } else if (index == 2) {
                                            //             return {
                                            //                 type: 'text',
                                            //                 content: 'delete',
                                            //             }
                                            //         }
                                            //     },
                                            //     onItemPress: (index) => {
                                            //         if (index == 0) {
                                            //             ask(cache["workspaceId"], { type: 'files.create', isDir: false, docTitle: 'hello.js', docPath: doc.path + "/" + doc.id }, (docs) => {
                                            //                 cache["docs"] = docs;
                                            //                 buildDocsTree();
                                            //                 updateApp(comp());
                                            //             });
                                            //         } else if (index == 1) {
                                            //             ask(cache["workspaceId"], { type: 'files.create', isDir: true, docTitle: 'hello.js', docPath: doc.path + "/" + doc.id }, (docs) => {
                                            //                 cache["docs"] = docs;
                                            //                 buildDocsTree();
                                            //                 updateApp(comp());
                                            //             });
                                            //         } else if (index == 2) {
                                            //             ask(cache["workspaceId"], { type: 'files.delete', docId: doc.id }, (docs) => {
                                            //                 cache["docs"] = docs;
                                            //                 buildDocsTree();
                                            //                 updateApp(comp());
                                            //             });
                                            //         }
                                            //     }
                                            // };
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
                        height: meta.height
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
                                        type: 'array',
                                        orientation: 'vertical',
                                        items: [
                                            {
                                                type: 'expanded',
                                                child: {
                                                    type: 'scroller',
                                                    child: {
                                                        type: 'array',
                                                        orientation: 'vertical',
                                                        items: cache["messages"].map(msg => {
                                                            return {
                                                                type: 'array',
                                                                orientation: 'horizontal',
                                                                margin: {
                                                                    top: 16
                                                                },
                                                                items: [
                                                                    {
                                                                        type: 'spacer'
                                                                    },
                                                                    {
                                                                        type: 'container',
                                                                        bgcolor: meta.primaryColor2,
                                                                        width: 220,
                                                                        borderRadius: 16,
                                                                        padding: {
                                                                            left: 8,
                                                                            top: 8,
                                                                            right: 8,
                                                                            bottom: 8
                                                                        },
                                                                        child: {
                                                                            type: 'text',
                                                                            content: msg,
                                                                            textColor: '#ffffff'
                                                                        }
                                                                    }
                                                                ]
                                                            };
                                                        })
                                                    },
                                                },
                                            },
                                            {
                                                type: 'input',
                                                key: 'chatInput',
                                                hint: 'type your prompt',
                                                onChange: (text) => {
                                                    cache["chatMessageInput"] = text;
                                                },
                                                trailing: {
                                                    type: 'container',
                                                    height: 52,
                                                    width: 92,
                                                    padding: {
                                                        left: 8,
                                                        top: 8,
                                                        right: 8,
                                                        bottom: 8
                                                    },
                                                    child: {
                                                        type: 'button',
                                                        label: 'Send',
                                                        onPress: () => {
                                                            if (!cache["chatMessageInput"] || cache["chatMessageInput"].length == 0) {
                                                                return;
                                                            }
                                                            if (!cache["messages"]) {
                                                                cache["messages"] = [];
                                                            }
                                                            cache["messages"].push(cache["chatMessageInput"]);
                                                            cache["chatMessageInput"] = "";
                                                            clearInput('chatInput');
                                                            updateApp(comp());
                                                        }
                                                    }
                                                }
                                            }
                                        ]
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
    let root = { path: '', title: 'src', id: '0', children: {}, key: '0', data: JSON.stringify({ path: '', title: 'src', id: '0' }) };
    cache["docs"].map((doc, index) => {
        let pathParts = doc.Path.split("/").slice(1);
        let temp = root;
        let progressPath = temp.id;
        for (let part in pathParts) {
            if (!temp[part]) {
                temp[part] = { path: progressPath, title: '', id: part, key: part, children: {} };
            }
            temp = temp[part].children;
            progressPath += '/' + part;
        }
        if (temp[doc.Id]) {
            temp[doc.Id].title = doc.Title;
            temp[doc.Id].data = JSON.stringify({ path: temp[doc.Id].path, title: doc.Title, id: temp[doc.Id].id });
        } else {
            temp[doc.Id] = { path: doc.Path, title: doc.Title, id: doc.Id, children: {}, key: doc.Id, data: JSON.stringify({ path: doc.Path, title: doc.Title, id: doc.Id }) };
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
    cache["docs"] = [{ children: [], key: '0', data: "src" }];
    cache["docsTree"] = { children: [], key: '0', data: "src" };
    cache["messages"] = [];
    ask(meta.pointId, { type: 'initWorkspace' }, (workspace) => {
        cache["workspaceId"] = workspace.Id;
        initApp(comp());
        // ask(cache["workspaceId"], { type: 'files.read' }, (docs) => {
        //     cache["docs"] = docs;
        //     buildDocsTree();
        //     updateApp(comp());
        // });
    });
} else {
    updateApp(comp());
}