import { useEffect, useRef, useState } from "react";
import { api } from "@/index";
import { colors } from "@nextui-org/theme";
import { Actions, States, useTheme } from "@/api/client/states";
import * as RGL from "react-grid-layout";
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import IconButton from "../elements/icon-button";
import AppletHost from "./applet-host";
import { openAppletSheet } from "./applet-sheet";
import { Spinner } from "@nextui-org/react";
import { App, Machine, MachineMeta } from "@/api/sigma/models";

const ResponsiveReactGridLayout = RGL.WidthProvider(RGL.Responsive);
export const rowHeight = 8
export const columnsDict: { [id: string]: number } = { lg: 14, md: 12, sm: 10, xs: 6, xxs: 4 }
export let sizeKey = window.innerWidth >= 1200 ? 'lg' : window.innerWidth >= 996 ? 'md' : window.innerWidth >= 768 ? 'sm' : window.innerWidth >= 480 ? 'xs' : 'xxs'
window.onresize = () => {
    sizeKey = window.innerWidth >= 1200 ? 'lg' : window.innerWidth >= 996 ? 'md' : window.innerWidth >= 768 ? 'sm' : window.innerWidth >= 480 ? 'xs' : 'xxs'
}

let buildLayoutOfWorkers = (workers: any[]) => {
    return {
        lg: workers.map((w: any) => ({ ...w.secret.grid.lg, i: w.id, static: false })),
        md: workers.map((w: any) => ({ ...w.secret.grid.md, i: w.id, static: false })),
        sm: workers.map((w: any) => ({ ...w.secret.grid.sm, i: w.id, static: false })),
        xs: workers.map((w: any) => ({ ...w.secret.grid.xs, i: w.id, static: false })),
        xxs: workers.map((w: any) => ({ ...w.secret.grid.xxs, i: w.id, static: false }))
    }
}

const measureWidgetSize = (bots: { [id: string]: any }, worker: any) => {
    let rowWorkersCount = Object.values(bots).filter(w => {
        return (
            (
                w.secret.grid[sizeKey].y >= worker.secret.grid[sizeKey].y &&
                w.secret.grid[sizeKey].y <= (worker.secret.grid[sizeKey].y + worker.secret.grid[sizeKey].h - 1)
            ) || (
                worker.secret.grid[sizeKey].y >= w.secret.grid[sizeKey].y &&
                worker.secret.grid[sizeKey].y <= (w.secret.grid[sizeKey].y + w.secret.grid[sizeKey].h - 1)
            )
        )
    }).length
    let unit = (window.innerWidth - (rowWorkersCount * 16)) / columnsDict[sizeKey]
    const widSize = {
        w: worker.secret.grid[sizeKey].w,
        width: worker.secret.grid[sizeKey].w * unit,
        h: worker.secret.grid[sizeKey].h,
        height: worker.secret.grid[sizeKey].h * rowHeight
    }
    return widSize
}

let cachedMembers: { [id: string]: any } = {};

export let addWidgetToSDesktop = async (_pointId: string, _appId: string, _machineId: string): Promise<any> => { };

const Desk = (props: { show: boolean, point: any }) => {

    const deskLayout = useRef<any>({});
    const [loadDesktop, setLoadDesktop] = useState(false)
    const editMode = States.useListener(States.store.boardEditingMode)
    const { theme, wallpaper } = useTheme();
    const metadataRef: any = useRef({})
    const [, setTrigger] = useState(Math.random().toString());
    const rerender = () => {
        setTrigger(Math.random().toString());
    }
    addWidgetToSDesktop = async (pointId: string, appId: string, machineId: string): Promise<any> => {
        let workersMax = 0
        let workers = Object.values(cachedMembers);
        if (workers.length > 0) {
            workersMax = Math.max(...workers.map(w => w.secret.grid[sizeKey].y + w.secret.grid[sizeKey].h)) + 1
        }
        let unit = window.innerWidth / columnsDict[sizeKey] - 16
        let url = "";
        if (machineId === "5c23a6dea8c7e58ec93459e85bb64de8") {
            let str = prompt("input the url to embed:");
            if (str) {
                url = str;
            }
        }
        let md = {
            grid: {
                'lg': { x: 0, y: workersMax, w: 2, h: unit / 8 },
                'md': { x: 0, y: workersMax, w: 2, h: unit / 8 },
                'sm': { x: 0, y: workersMax, w: 2, h: unit / 8 },
                'xs': { x: 0, y: workersMax, w: 2, h: unit / 8 },
                'xxs': { x: 0, y: workersMax, w: 2, h: unit / 8 }
            },
            frameUrl: url
        }
        let identifier = window.crypto.randomUUID();
        await api.services?.points.addMachine(pointId, appId,
            new MachineMeta(
                machineId,
                md,
                identifier,
                new Map(),
            )
        );
        let m = {
            metadata: md,
            identifier: identifier,
            machineId: machineId
        };
        cachedMembers[machineId + "::" + identifier] = m;
        setTimeout(() => {
            ['lg', 'md', 'sm', 'xs', 'xxs'].forEach(cat => {
                deskLayout.current[cat].push({ ...(m.metadata.grid as any)[cat], i: m.machineId + "::" + m.identifier, static: false })
            })
            rerender();
            api.services?.points.signal(pointId, m.machineId, 'single', JSON.stringify({ tag: 'get/widget', theme, widgetSize: measureWidgetSize(cachedMembers, m), secondaryColor: colors.purple, colorName: "blue", colors: colors.blue }));
        });
    }
    useEffect(() => {
        let packetReceiver = api.services?.addListener('points/signal', (packet: any) => {
            let data = JSON.parse(packet.data);
            metadataRef.current[packet.member.id] = { onClick: data.onClick }
            if (data.tag === 'get/widget') {
                console.log(cachedMembers);
                if (cachedMembers[packet.member.id]) {
                    cachedMembers[packet.member.id].code = data.code;
                    rerender();
                }
            }
        });
        setTimeout(() => {
            setLoadDesktop(true)
        }, 750);
        return () => {
            api.services?.delListener('points/signal', packetReceiver ?? "");
        }
    }, []);

    useEffect(() => {
        api.services?.points.listApps(props.point.id).then((apps: App[]) => {
            let filtered: { [id: string]: any } = {};
            cachedMembers = {};
            apps.forEach((a: App) => {
                a.machines.forEach((m: Machine) => {
                    if (m.identifier != '0') {
                        filtered[m.id + "::" + m.identifier] = m;
                        cachedMembers[m.id + "::" + m.identifier] = m;
                    }
                });
            });
            deskLayout.current = buildLayoutOfWorkers(Object.values(filtered));
            rerender();
            setTimeout(() => {
                Object.values(filtered).forEach(worker => {
                    api.services?.points.signal(props.point.id, worker.id, 'single', JSON.stringify({ tag: 'get/widget', theme, widgetSize: measureWidgetSize(cachedMembers, worker), secondaryColor: colors.purple, colorName: "blue", colors: colors.blue }));
                })
            });
        })
    }, [props.point.id]);

    useEffect(() => {
        AppletHost.unloadAllHosts();
        Object.values(cachedMembers).forEach(worker => {
            api.services?.points.signal(props.point.id, worker.id, 'single', JSON.stringify({ tag: 'get/widget', theme, widgetSize: measureWidgetSize(cachedMembers, worker), secondaryColor: colors.purple, colorName: "blue", colors: colors.blue }));
        })
    }, [theme]);

    return (
        <div
            style={{ width: '100%', position: 'relative', paddingBottom: 200 }}
        >
            {
                deskLayout.current.lg ?
                    <ResponsiveReactGridLayout
                        key={props.point.id}
                        className="layout"
                        style={{ minHeight: 'auto', paddingTop: wallpaper === "true" ? 56 : 56, width: window.innerWidth, minWidth: window.innerWidth + 'px', display: loadDesktop ? 'block' : 'hidden' }}
                        cols={{ lg: 14, md: 12, sm: 10, xs: 6, xxs: 4 }}
                        rowHeight={rowHeight}
                        width={window.innerWidth}
                        layouts={structuredClone(deskLayout.current)}
                        isDraggable={editMode}
                        isResizable={editMode}
                        draggableCancel=".cancelSelectorName"
                        draggableHandle=".drag-handle"
                        breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480, xxs: 0 }}
                        onLayoutChange={(_: RGL.Layout[], layouts: RGL.Layouts) => {
                            const oldLayouts = deskLayout.current;
                            let clonedLayouts = structuredClone(layouts)
                            for (let sizeKey in oldLayouts) {
                                for (let i = 0; i < oldLayouts[sizeKey].length; i++) {
                                    if (JSON.stringify(oldLayouts[sizeKey][i]) !== JSON.stringify(clonedLayouts[sizeKey][i])) {
                                        let md = JSON.parse(cachedMembers[oldLayouts[sizeKey][i].i].metadata);
                                        ['lg', 'md', 'sm', 'xs', 'xxs'].forEach(cat => {
                                            let data = clonedLayouts[sizeKey][i];
                                            md.secret.grid[cat] = {
                                                x: data.x, y: data.y, w: data.w, h: data.h
                                            };
                                        });
                                        let memberId: string = oldLayouts[sizeKey][i].i;
                                        cachedMembers[memberId].metadata = JSON.stringify(md);
                                        cachedMembers[memberId].secret = md.secret;
                                        api.services?.points.updateMachine(
                                            props.point.id,
                                            cachedMembers[memberId].appId,
                                            new MachineMeta(
                                                memberId.split("::")[0],
                                                md,
                                                memberId.split("::")[1],
                                                new Map(),
                                            ),
                                        );
                                    }
                                }
                            }
                            deskLayout.current = clonedLayouts;
                        }}
                    >
                        {
                            deskLayout.current[window.innerWidth >= 1200 ? 'lg' : window.innerWidth >= 996 ? 'md' : window.innerWidth >= 768 ? 'sm' : window.innerWidth >= 480 ? 'xs' : 'xxs'].map((item: any) => item.i).map((key: string, index: number) => {
                                return (
                                    <div key={key} style={{ overflow: 'hidden', borderRadius: 4 }} data-grid={deskLayout.current[window.innerWidth >= 1200 ? 'lg' : window.innerWidth >= 996 ? 'md' : window.innerWidth >= 768 ? 'sm' : window.innerWidth >= 480 ? 'xs' : 'xxs'][index]}>
                                        <div className="w-full h-full" style={{ position: 'relative' }}>
                                            {
                                                (cachedMembers[key]?.id === "5c23a6dea8c7e58ec93459e85bb64de8") ? (
                                                    <iframe
                                                        frameBorder={'none'}
                                                        src={cachedMembers[key]?.secret?.frameUrl}
                                                        style={{
                                                            width: '100%',
                                                            height: '100%'
                                                        }}
                                                    />
                                                ) : (
                                                    <AppletHost.Host
                                                        isWidget
                                                        appletKey={key}
                                                        onClick={() => {
                                                            if (!editMode) {
                                                                let onClickOfMetadata = metadataRef.current[key]?.onClick
                                                                if (onClickOfMetadata) {
                                                                    Actions.updateOverlayData({ code: onClickOfMetadata.code, workerId: key, room: props.point })
                                                                } else {
                                                                    openAppletSheet(props.point, key)
                                                                }
                                                            }
                                                        }}
                                                        entry={cachedMembers[key]?.code ? 'Test' : 'Dummy'}
                                                        code={
                                                            cachedMembers[key]?.code ?? 'class Dummy { constructor() {} onMount() {} render() { return "" } }'
                                                        }
                                                        index={index}
                                                    />
                                                )
                                            }
                                            {
                                                editMode ? (
                                                    <div className="w-full h-full" style={{ position: 'absolute', left: 0, top: 0 }} />
                                                ) : null
                                            }
                                            {
                                                editMode ? (
                                                    <IconButton
                                                        name="delete"
                                                        size={[16, 16]}
                                                        style={{ transform: 'translate(8px, -68px)' }}
                                                        onClick={() => {
                                                            if (window.confirm('do you want to delete this widget ?')) {
                                                                api.services?.points.removeMachine(props.point.id, cachedMembers[key].appId, key.split("::")[0], key.split("::")[1]).then(() => {
                                                                    Object.keys(deskLayout.current).forEach(lk => {
                                                                        deskLayout.current[lk] = deskLayout.current[lk].filter((item: any) => item.i !== key);
                                                                    })
                                                                    delete cachedMembers[key];
                                                                    rerender();
                                                                })
                                                            }
                                                        }}
                                                        className="cancelSelectorName" />
                                                ) : null
                                            }
                                            {
                                                editMode ? (
                                                    <IconButton
                                                        name="drag"
                                                        size={[16, 16]}
                                                        className="drag-handle"
                                                        style={{ transform: 'translate(16px, -68px)' }} />
                                                ) : null
                                            }
                                        </div>
                                    </div>
                                )
                            })
                        }
                    </ResponsiveReactGridLayout >
                    : null
            }
            {
                loadDesktop ?
                    null : (
                        <div style={{ width: '100%', height: '100%', position: 'absolute', left: '50%', top: 'calc(50% - 32px)', transform: 'translate(-50%, -50%)' }}>
                            <Spinner style={{
                                width: 40,
                                height: 40,
                                position: 'absolute',
                                left: '50%',
                                top: '50%',
                                transform: 'translate(-50%, -50%)'
                            }} />
                        </div>
                    )
            }
        </div>
    )
}

export default Desk
