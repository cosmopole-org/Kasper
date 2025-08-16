try {
    function comp() {
        return {
            root: {
                type: "container",
                args: {
                    height: meta.height,
                    width: meta.width,
                    padding: 2,
                    decoration: {
                        color: '#FFF',
                        "border": {
                            "color": "#FFF06292",
                            "width": 4
                        },
                        "shape": "circle"
                    },
                    child: {
                        type: 'stack',
                        args: {
                            children: [
                                {
                                    type: 'positioned',
                                    args: {
                                        left: 0,
                                        top: 0,
                                        width: meta.width - 12,
                                        height: meta.height - 12,
                                        child: {
                                            "type": "clip_rrect",
                                            "args": {
                                                "borderRadius": {
                                                    "type": "circular",
                                                    "radius": (meta.width - 8) / 2
                                                },
                                                "child": {
                                                    "type": "cors_image",
                                                    "args": {
                                                        "url": "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRHbVouA1lqUuIJkwgtdfg2ahZTQ2FCDOscSQ&s"
                                                    }
                                                }
                                            }
                                        }
                                    }
                                },
                                {
                                    type: 'positioned',
                                    args: {
                                        left: 0,
                                        top: 0,
                                        width: meta.width - 8,
                                        height: meta.height - 8,
                                        child: {
                                            type: "center",
                                            args: {
                                                child: {
                                                    "type": "rotator",
                                                    "args": {
                                                        "angle": '\${hoursAngle}',
                                                        child: {
                                                            type: "container",
                                                            args: {
                                                                width: meta.width - 72,
                                                                height: 12,
                                                                child: {
                                                                    type: 'stack',
                                                                    args: {
                                                                        children: [
                                                                            {
                                                                                type: 'positioned',
                                                                                args: {
                                                                                    left: (meta.width - 72) / 2,
                                                                                    child: {
                                                                                        type: 'container',
                                                                                        args: {
                                                                                            width: (meta.width - 72) / 2,
                                                                                            height: 12,
                                                                                            color: '#f90'
                                                                                        }
                                                                                    }
                                                                                }
                                                                            }
                                                                        ]
                                                                    }
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                },
                                {
                                    type: 'positioned',
                                    args: {
                                        left: 0,
                                        top: 0,
                                        width: meta.width - 8,
                                        height: meta.height - 8,
                                        child: {
                                            type: "center",
                                            args: {
                                                child: {
                                                    "type": "rotator",
                                                    "args": {
                                                        "angle": '\${minutesAngle}',
                                                        child: {
                                                            type: "container",
                                                            args: {
                                                                width: meta.width - 56,
                                                                height: 8,
                                                                child: {
                                                                    type: 'stack',
                                                                    args: {
                                                                        children: [
                                                                            {
                                                                                type: 'positioned',
                                                                                args: {
                                                                                    left: (meta.width - 56) / 2,
                                                                                    child: {
                                                                                        type: 'container',
                                                                                        args: {
                                                                                            width: (meta.width - 56) / 2,
                                                                                            height: 8,
                                                                                            color: '#09f'
                                                                                        }
                                                                                    }
                                                                                }
                                                                            }
                                                                        ]
                                                                    }
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                },
                                {
                                    type: 'positioned',
                                    args: {
                                        left: 0,
                                        top: 0,
                                        width: meta.width - 8,
                                        height: meta.height - 8,
                                        child: {
                                            type: "center",
                                            args: {
                                                child: {
                                                    "type": "rotator",
                                                    "args": {
                                                        "angle": '\${secondsAngle}',
                                                        child: {
                                                            type: "container",
                                                            args: {
                                                                width: meta.width - 40,
                                                                height: 4,
                                                                child: {
                                                                    type: 'stack',
                                                                    args: {
                                                                        children: [
                                                                            {
                                                                                type: 'positioned',
                                                                                args: {
                                                                                    left: (meta.width - 40) / 2,
                                                                                    child: {
                                                                                        type: 'container',
                                                                                        args: {
                                                                                            width: (meta.width - 40) / 2,
                                                                                            height: 4,
                                                                                            color: '#09f'
                                                                                        }
                                                                                    }
                                                                                }
                                                                            }
                                                                        ]
                                                                    }
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            ]
                        }
                    }
                }
            }
        };
    }
    if (!started) {
        updateProp('hoursAngle', (30 * (new Date()).getHours()) - 90);
        updateProp('minutesAngle', (6 * (new Date()).getMinutes()) - 90);
        updateProp('secondsAngle', (6 * (new Date()).getSeconds()) - 90);
        initApp(comp());
        setInterval(() => {
            updateProp('hoursAngle', (30 * (new Date()).getHours()) - 90);
            updateProp('minutesAngle', (6 * (new Date()).getMinutes()) - 90);
            updateProp('secondsAngle', (6 * (new Date()).getSeconds()) - 90);
        }, 1000);
    } else {
        updateProp('hoursAngle', (30 * (new Date()).getHours()) - 90);
        updateProp('minutesAngle', (6 * (new Date()).getMinutes()) - 90);
        updateProp('secondsAngle', (6 * (new Date()).getSeconds()) - 90);
    }
} catch (ex) {
    log(ex.toString());
}