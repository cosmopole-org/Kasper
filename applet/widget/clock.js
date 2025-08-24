try {
    function comp(hoursAngle, minutesAngle, secondsAngle) {
        return {
            root: {
                type: "glassContainer",
                borderRadius: meta.width / 2,
                child: {
                    type: "container",
                    height: meta.height,
                    width: meta.width,
                    shape: "circle",
                    borderColor: "#999999",
                    child: {
                        type: 'freelayout',
                        items: [
                            {
                                type: "container",
                                left: 0,
                                top: 0,
                                width: meta.width,
                                height: meta.height,
                                child: {
                                    type: 'freelayout',
                                    items: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11].map((index) => (
                                        {
                                            type: "container",
                                            left: 0,
                                            top: 0,
                                            width: meta.width,
                                            height: meta.height,
                                            child: {
                                                type: "container",
                                                center: true,
                                                transform: {
                                                    rotation: index * 30 - 90
                                                },
                                                child: {
                                                    type: "container",
                                                    width: meta.width,
                                                    height: 24,
                                                    child: {
                                                        type: 'freelayout',
                                                        items: [
                                                            {
                                                                type: "container",
                                                                left: meta.width - 32,
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
                                width: meta.width,
                                height: meta.height,
                                child: {
                                    type: "container",
                                    center: true,
                                    transform: {
                                        rotation: hoursAngle
                                    },
                                    child: {
                                        type: "container",
                                        width: meta.width * 3 / 5,
                                        height: 12,
                                        child: {
                                            type: 'freelayout',
                                            items: [
                                                {
                                                    type: "container",
                                                    left: (meta.width * 3 / 5) / 2,
                                                    width: (meta.width * 3 / 5) / 2,
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
                                width: meta.width,
                                height: meta.height,
                                child: {
                                    type: "container",
                                    center: true,
                                    transform: {
                                        rotation: minutesAngle
                                    },
                                    child: {
                                        type: "container",
                                        width: meta.width * 3.5 / 5,
                                        height: 8,
                                        child: {
                                            type: 'freelayout',
                                            items: [
                                                {
                                                    type: "container",
                                                    left: (meta.width * 3.5 / 5) / 2,
                                                    width: (meta.width * 3.5 / 5) / 2,
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
                                width: meta.width,
                                height: meta.height,
                                child: {
                                    type: "container",
                                    center: true,
                                    transform: {
                                        rotation: secondsAngle
                                    },
                                    child: {
                                        type: "container",
                                        width: meta.width * 4 / 5,
                                        height: 4,
                                        child: {
                                            type: 'freelayout',
                                            items: [
                                                {
                                                    type: "container",
                                                    left: (meta.width * 4 / 5) / 2,
                                                    width: (meta.width * 4 / 5) / 2,
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
        };
    }
    if (!started) {
        initApp(comp((30 * (new Date()).getHours()) - 90, (6 * (new Date()).getMinutes()) - 90, (6 * (new Date()).getSeconds()) - 90));
        setInterval(() => {
            updateApp(comp((30 * (new Date()).getHours()) - 90, (6 * (new Date()).getMinutes()) - 90, (6 * (new Date()).getSeconds()) - 90));
        }, 1000);
    } else {
        updateApp(comp((30 * (new Date()).getHours()) - 90, (6 * (new Date()).getMinutes()) - 90, (6 * (new Date()).getSeconds()) - 90));
    }
} catch (ex) {
    log(ex.toString());
}