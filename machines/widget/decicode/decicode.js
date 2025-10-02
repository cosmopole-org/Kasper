try {
    function comp() {
        return {
            root: {
                type: "glassContainer",
                borderRadius: meta.width / 5,
                child: {
                    type: "container",
                    height: meta.height,
                    width: meta.width,
                    child: {
                        type: 'button',
                        buttonStyle: 'elevated',
                        label: "Deci Code",
                        bgcolor: meta.primaryColor1,
                        onPress: () => {
                            openWindow("applet", 1700, 900);
                        }
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