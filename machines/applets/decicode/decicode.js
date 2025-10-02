try {
    function comp() {
        return {
            root: {
                type: "glassContainer",
                child: {
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
                                height: meta.height,
                                child: {
                                    type: 'treeView',
                                    treeData: {
                                        key: 'node1',
                                        children: [
                                            {
                                                key: 'node2',
                                                children: []
                                            },
                                            {
                                                key: 'node3',
                                                children: []
                                            }
                                        ]
                                    },
                                    itemBuilder: (key, level) => {
                                        return {
                                            type: 'text',
                                            content: level == 0 ? "src" : key
                                        };
                                    },
                                    onItemTap: (key) => {
                                        log(key + " tapped !");
                                    }
                                }
                            },
                            {
                                type: 'codeEditor',
                                key: "mainCode",
                                minLines: 35,
                                width: meta.width - 250 - 300,
                                height: meta.height
                            },
                            {
                                type: "container",
                                width: 32,
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
                    }
                }
            }
        };
    }
    if (!started) {
        initApp(comp());
    } else {
        updateApp(comp());
    }
} catch (ex) {
    log(ex.toString());
}