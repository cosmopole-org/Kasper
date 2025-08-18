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
                                type: "image",
                                left: 0,
                                top: 0,
                                opacity: 0.5,
                                clip: 'oval',
                                width: meta.width,
                                height: meta.height,
                                url: "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRHbVouA1lqUuIJkwgtdfg2ahZTQ2FCDOscSQ&s"
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
                                                    bgcolor: '#FF9900',
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
                                                    bgcolor: '#0099FF',
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
                                                    bgcolor: '#FF99FF',
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