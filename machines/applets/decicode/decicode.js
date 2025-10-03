
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
                            type: 'treeView',
                            treeData: {
                                key: 'root',
                                data: 'src',
                                children: [
                                    {
                                        key: 'resources',
                                        data: 'resources',
                                        children: [
                                            {
                                                key: 'background',
                                                data: 'background.png',
                                                children: []
                                            },
                                            {
                                                key: 'logo',
                                                data: 'logo.png',
                                                children: []
                                            }
                                        ]
                                    },
                                    {
                                        key: 'logic',
                                        data: 'logic',
                                        children: [
                                            {
                                                key: 'sdk',
                                                data: 'sdk',
                                                children: [
                                                    {
                                                        key: 'api',
                                                        data: 'api.js',
                                                        children: []
                                                    },
                                                    {
                                                        key: 'constants',
                                                        data: 'costants.json',
                                                        children: []
                                                    },
                                                ]
                                            },
                                            {
                                                key: 'widgetjs',
                                                data: 'widget.js',
                                                children: []
                                            },
                                            {
                                                key: 'applet',
                                                data: 'applet.js',
                                                children: []
                                            },
                                        ]
                                    }
                                ]
                            },
                            itemBuilder: (key, data, level) => {
                                return {
                                    type: 'text',
                                    content: level == 0 ? "src" : data
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
                                                    type: 'button',
                                                    buttonStyle: 'elevated',
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
if (!started) {
    cache["messages"] = [];
    initApp(comp());
} else {
    updateApp(comp());
}