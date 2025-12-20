
function openTheApplet() {
    openWindow("applet", 1200, 800);
}

function comp() {
    return {
        type: "clipRRect",
        borderRadius: 24,
        clipBehavior: "antiAlias",
        child: {
            type: "glassContainer",
            child: {
                type: "container",
                height: meta.height,
                width: meta.width,
                padding: {
                    left: 8, top: 8, right: 8, bottom: 8
                },
                child: {
                    type: 'elevatedButton',
                    child: {
                        type: "text",
                        data: "Deci Code"
                    },
                    style: {
                        backgroundColor: "primary"
                    },
                    onPressed: { actionType: "event", appId: meta.appId, key: "openTheApplet", args: [] }
                }
            }
        }
    };
}
if (!started) {
    render(comp());
} else {
    render(comp());
}