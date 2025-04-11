import { Button, Card } from "@nextui-org/react";
import React from "react";
import { DrawerItemProps } from "@/types";
import Icon from "./icon";
import { Logo } from "../icons";
import { usePathname, useRouter } from "next/navigation";
import { useHookstate } from "@hookstate/core";
import { selectedGame, selectedMode } from "../navbar";
import { clearHall } from "@/api/online/messenger";
import { clearReports } from "@/api/online/report";
import { clearMessages } from "../hall/chat";
import { clearReps } from "../reports/list";

export default function Drawer({ items, open, onOpenStateChange }: Readonly<{ items?: DrawerItemProps[], open?: boolean, onOpenStateChange?: (open: boolean) => void }>) {
  const router = useRouter();
  const path = usePathname();
  const game = useHookstate(selectedGame).get({ noproxy: true });
  const mode = useHookstate(selectedMode).get({ noproxy: true });
  return (
    <Card className={(open ? 'w-72' : 'w-20') + ` h-full rounded-none pl-[14px] pr-[14px] pt-5 gap-4 fixed`}
      style={{ backgroundColor: '#581c87', transition: 'width 250ms' }}>
      <Button size="lg" className={open ? 'ml-[100px] mb-4 bg-white' : 'mb-4 bg-white'} isIconOnly onClick={() => {
        if (onOpenStateChange) onOpenStateChange(!open)
      }}>
        <Logo color="#000000" />
      </Button>
      {
        items ?
          items.map(item => {
            const isFilled = item.variant === "filled";
            return (
              <Button onClick={() => router.push(item.action + `?gameKey=${game}&mode=${mode}`)} size="lg" key={item.key} isIconOnly={!open} className={isFilled ? 'mb-2 bg-white' : 'bg-transparent'}>
                <Icon color={isFilled ? '#000000' : '#ffffff'} name={item.icon ?? ""} size={[28, 28]} iconType={item.iconType} />
                <div className={"flex flex-col " + ((isFilled || !open) ? 'text-center ' : 'flex-1 text-left ')}>
                  <span
                    className={((isFilled || !open) ? 'text-center ' : 'flex-1 text-left ')}
                    style={{ color: isFilled ? '#000' : '#fff' }}>
                    {open ? item.title : null}
                  </span>
                  {open ? <span className="text-sm text-default-400 text-left">{item.subtitle}</span> : null}
                </div>
              </Button>
            )
          }) :
          null
      }
      {
        ["/home/hall", "/home/reports"].includes(path) ? (
          <Button size="lg" className={'fixed bottom-4 bg-white'} isIconOnly onClick={() => {
            if (path == "/home/hall") {
              clearHall().then(() => clearMessages());
            } else if (path == "/home/reports") {
              clearReports().then(() => clearReps());
            }
          }}>
            <Icon name="delete" color="#000000" />
          </Button>
        ) : null
      }
    </Card>
  )
}
