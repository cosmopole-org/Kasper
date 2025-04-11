import { SVGProps } from "react";

export type IconSvgProps = SVGProps<SVGSVGElement> & {
  size?: number;
};

export type NavbarItemProps = { key: string, title?: string, icon?: string, action?: string, compType?: string }

export type IconType = 'circle'

export type DrawerItemProps = { key: string, title?: string, subtitle?: string, icon?: string, action: string, variant?: string, iconType?: IconType }

export type Variable = { key: string, value: any, type: string }

export type Player = { id: string, playerName: string, email: string, coin: number, gem: number, energy: number }

export type LbPlayer = { userId: string, name: string, avatar: number, score: any }
