
function comp(hoursAngle, minutesAngle, secondsAngle) {
    return {
        type: "clipRRect",
        borderRadius: 9999,
        clipBehavior: "antiAlias",
        child: {
            type: "glassContainer",
            child: {
                type: "container",
                height: meta.height,
                width: meta.width,
                child: {
                    type: 'stack',
                    children: [
                        {
                            type: "positioned",
                            left: 0,
                            top: 0,
                            width: meta.width,
                            height: meta.height,
                            child: {
                                type: 'stack',
                                children: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11].map((index) => (
                                    {
                                        type: "positioned",
                                        left: 0,
                                        top: 0,
                                        width: meta.width,
                                        height: meta.height,
                                        child: {
                                            type: "transform",
                                            action: 'rotate',
                                            data: index * 30 - 90,
                                            child: {
                                                type: "container",
                                                width: meta.width,
                                                height: 24,
                                                child: {
                                                    type: 'stack',
                                                    "alignment": "center",
                                                    "clipBehavior": "antiAlias",
                                                    "fit": "expand",
                                                    "textDirection": "ltr",
                                                    children: [
                                                        {
                                                            type: "positioned",
                                                            left: meta.width - 32,
                                                            child: {
                                                                type: "transform",
                                                                action: 'rotate',
                                                                data: -(index * 30 - 90),
                                                                child: {
                                                                    type: "text",
                                                                    data: (index == 0 ? 12 : index).toString(),
                                                                    "style": {
                                                                        "color": meta.textColor,
                                                                        fontSize: 20,
                                                                    }
                                                                }
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
                            type: "positioned",
                            left: 0,
                            top: 0,
                            width: meta.width,
                            height: meta.height,
                            child: {
                                type: "transform",
                                action: 'rotate',
                                data: hoursAngle,
                                child: {
                                    type: "container",
                                    width: meta.width * 3 / 5,
                                    height: 12,
                                    child: {
                                        type: 'stack',
                                        "alignment": "center",
                                        "clipBehavior": "antiAlias",
                                        "fit": "expand",
                                        "textDirection": "ltr",
                                        children: [
                                            {
                                                type: "positioned",
                                                left: (meta.width * 3 / 5)  / 2 + 32,
                                                width: (meta.width * 3 / 5) / 2,
                                                height: 12,
                                                child: {
                                                    type: 'container',
                                                    decoration: {
                                                        color: meta.primaryColor1,
                                                        borderRadius: 6
                                                    }
                                                }
                                            }
                                        ]
                                    }
                                }
                            }
                        },
                        {
                            type: "positioned",
                            left: 0,
                            top: 0,
                            width: meta.width,
                            height: meta.height,
                            child: {
                                type: "transform",
                                action: 'rotate',
                                data: minutesAngle,
                                child: {
                                    type: "container",
                                    width: meta.width * 3.5 / 5,
                                    height: 8,
                                    child: {
                                        type: 'stack',
                                        "alignment": "center",
                                        "clipBehavior": "antiAlias",
                                        "fit": "expand",
                                        "textDirection": "ltr",
                                        children: [
                                            {
                                                type: "positioned",
                                                left: (meta.width * 3.5 / 5) / 2 + 32,
                                                width: (meta.width * 3.5 / 5) / 2,
                                                height: 8,
                                                child: {
                                                    type: 'container',
                                                    decoration: {
                                                        color: meta.primaryColor2,
                                                        borderRadius: 4
                                                    }
                                                }
                                            }
                                        ]
                                    }
                                }
                            }
                        },
                        {
                            type: "positioned",
                            left: 0,
                            top: 0,
                            width: meta.width,
                            height: meta.height,
                            child: {
                                type: "transform",
                                action: 'rotate',
                                data: secondsAngle,
                                child: {
                                    type: "container",
                                    width: meta.width * 4 / 5,
                                    height: 4,
                                    child: {
                                        type: 'stack',
                                        "alignment": "center",
                                        "clipBehavior": "antiAlias",
                                        "fit": "expand",
                                        "textDirection": "ltr",
                                        children: [
                                            {
                                                type: "positioned",
                                                left: (meta.width * 4 / 5) / 2 + 16,
                                                width: (meta.width * 4 / 5) / 2,
                                                height: 4,
                                                child: {
                                                    type: 'container',
                                                    decoration: {
                                                        color: meta.primaryColor3,
                                                        borderRadius: 2
                                                    }
                                                }
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
    };
}
if (!started) {
    render(comp((30 * (new Date()).getHours()) - 90, (6 * (new Date()).getMinutes()) - 90, (6 * (new Date()).getSeconds()) - 90));
    setInterval(() => {
        render(comp((30 * (new Date()).getHours()) - 90, (6 * (new Date()).getMinutes()) - 90, (6 * (new Date()).getSeconds()) - 90));
    }, 1000);
} else {
    render(comp((30 * (new Date()).getHours()) - 90, (6 * (new Date()).getMinutes()) - 90, (6 * (new Date()).getSeconds()) - 90));
}