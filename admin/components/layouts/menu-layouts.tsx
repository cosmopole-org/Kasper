"use client"

import { DrawerItemProps, NavbarItemProps } from "@/types";
import { Navbar } from "../navbar";
import Drawer from "../elements/drawer";
import { useState } from "react";
import { useTheme } from "next-themes";

export default function MenuLayout({ me, drawerItems, navItems, routeKey, children }: Readonly<{ me: any, drawerItems: DrawerItemProps[], navItems: NavbarItemProps[], routeKey: string, children?: any }>) {
    const [open, setOpen] = useState(false)
    const { theme } = useTheme()
    return (
        <div className="w-full h-full flex" style={{ backgroundColor: theme === 'light' ? '#f3e8ff' : '#111' }}>
            <Drawer open={open} onOpenStateChange={(o: boolean) => setOpen(o)} items={drawerItems} />
            <div className={`w-full h-full ${open ? 'ml-72' : 'ml-20'}`} style={{ transition: 'margin-left 250ms' }}>
                <Navbar routeKey={routeKey} items={navItems} me={me} />
                <div className="w-full flex overflow-hidden" style={{ height: 'calc(100% - 88px)' }}>
                    <div className="w-full h-full overflow-y-auto" style={{ maxWidth: '100%' }}>
                        {children}
                    </div>
                </div>
            </div>
        </div>
    )
}
