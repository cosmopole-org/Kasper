"use client"

import {
	Navbar as NextUINavbar,
	NavbarContent,
	NavbarMenu,
	NavbarMenuToggle,
	NavbarBrand,
	NavbarItem,
	NavbarMenuItem,
} from "@nextui-org/navbar";
import { Input, Link, Kbd, Dropdown, DropdownTrigger, Avatar, DropdownMenu, DropdownItem, Button, Selection } from "@nextui-org/react";

import { link as linkStyles } from "@nextui-org/theme";

import { siteConfig } from "@/config/site";
import NextLink from "next/link";
import clsx from "clsx";

import { ThemeSwitch } from "@/components/theme-switch";
import {
	Logo,
	GithubIcon,
	SearchIcon,
} from "@/components/icons";
import { NavbarItemProps } from "@/types";
import DropDownBox from "./elements/dropdown";
import { capitalize } from '../utils';
import { restart, signout } from "@/api/online/auth";
import { useRouter } from "next/navigation";
import Icon from "./elements/icon";
import { hookstate, useHookstate } from "@hookstate/core";
import { useEffect } from "react";
import { switchMode } from "@/api/store";

export const selectedGame = hookstate("cars");
export const selectedMode = hookstate("prod");

export const Navbar = ({ routeKey, items, me }: Readonly<{ me: any, routeKey: string, items: NavbarItemProps[] }>) => {
	const router = useRouter();
	const selectedGameState = useHookstate(selectedGame);
	const selectedModeState = useHookstate(selectedMode);
	useEffect(() => {
		const gameKey = window.location.toString().split("?")[1].split("&")[0].split("=")[1];
		const mode = window.location.toString().split("?")[1].split("&")[1].split("=")[1];
		selectedGameState.set(gameKey);
		switchMode(mode);
		selectedModeState.set(mode);
	}, []);
	const searchInput = (
		<Input
			aria-label="Search"
			classNames={{
				inputWrapper: "bg-default-100",
				input: "text-sm",
			}}
			endContent={
				<Kbd className="hidden lg:inline-block" keys={["command"]}>
					K
				</Kbd>
			}
			labelPlacement="outside"
			placeholder="Search..."
			startContent={
				<SearchIcon className="text-base text-default-400 pointer-events-none flex-shrink-0" />
			}
			type="search"
		/>
	);
	const gamesMenu = (
		<Dropdown backdrop="blur">
			<DropdownTrigger>
				<Button
					variant="bordered"
					className="capitalize"
					endContent={<Icon name="dropdown" size={[12, 12]} />}
				>
					<p className="mr-2">{selectedGameState.get({ noproxy: true })}</p>
				</Button>
			</DropdownTrigger>
			<DropdownMenu
				aria-label="Single selection example"
				variant="flat"
				disallowEmptySelection
				selectionMode="single"
				selectedKeys={selectedGameState.get({ noproxy: true })}
				onAction={(key) => {
					selectedGameState.set(key.toString());
					router.replace(`?gameKey=${key.toString()}&mode=${selectedModeState.get({ noproxy: true })}`);
				}}
			>
				<DropdownItem key="cars">Cars</DropdownItem>
				<DropdownItem key="hokm">Hokm</DropdownItem>
			</DropdownMenu>
		</Dropdown>
	);

	const serversMenu = (
		<Dropdown backdrop="blur">
			<DropdownTrigger>
				<Button
					variant="bordered"
					className="capitalize"
					endContent={<Icon name="dropdown" size={[12, 12]} />}
				>
					<p className="mr-2">{selectedModeState.get({ noproxy: true })}</p>
				</Button>
			</DropdownTrigger>
			<DropdownMenu
				aria-label="Single selection example"
				variant="flat"
				disallowEmptySelection
				selectionMode="single"
				selectedKeys={selectedModeState.get({ noproxy: true })}
				onAction={async (key) => {
					await switchMode(key.toString());
					selectedModeState.set(key.toString());
					router.replace(`?gameKey=${selectedGameState.get({ noproxy: true })}&mode=${key.toString()}`);
				}}
			>
				<DropdownItem key="prod">Producton</DropdownItem>
				<DropdownItem key="dev">Development</DropdownItem>
			</DropdownMenu>
		</Dropdown>
	);

	return (
		<NextUINavbar maxWidth="full" className="w-full pt-3 pb-3 shadow-md" position="sticky">
			<NavbarContent className="basis-1/5 sm:basis-full" justify="start">
				<NavbarBrand as="li" className="gap-3 max-w-fit">
					<NextLink className="flex justify-start items-center gap-1" href="/">
						<Logo />
						<p className="font-bold text-inherit">Midopia Admin</p>
					</NextLink>
				</NavbarBrand>
				<ul className="hidden lg:flex gap-8 justify-start ml-2">
					{items.map((item) => {
						if (item.compType === 'dropdown') {
							return (
								<NavbarItem key={routeKey + '-' + item.key} className="mt-1">
									<DropDownBox />
								</NavbarItem>
							)
						} else if (!item.compType || (item.compType === 'item')) {
							return (
								<NavbarItem key={routeKey + '-' + item.key} className="mt-3 mb-3">
									<NextLink
										className={clsx(
											linkStyles({ color: "foreground" }),
											"data-[active=true]:text-primary data-[active=true]:font-medium"
										)}
										color="foreground"
										href={item.action ?? ""}
									>
										{item.title}
									</NextLink>
								</NavbarItem>
							)
						} else {
							return null
						}
					})}
				</ul>
			</NavbarContent>

			<NavbarContent
				className="hidden sm:flex basis-1/5 sm:basis-full"
				justify="end"
			>
				{serversMenu}
				{gamesMenu}
				<NavbarItem className="hidden sm:flex gap-2">
					<ThemeSwitch />
				</NavbarItem>
				<NavbarItem className="hidden lg:flex">{searchInput}</NavbarItem>
				<Dropdown placement="bottom-end">
					<DropdownTrigger>
						<Avatar
							isBordered
							as="button"
							className="transition-transform"
							color="secondary"
							name={me.username.split("@")[0].substring(0, 1).toUpperCase()}
							size="lg"
						/>
					</DropdownTrigger>
					<DropdownMenu aria-label="Profile Actions" variant="flat">
						<DropdownItem key="profile" className="h-14 gap-2">
							<p className="">Signed in as</p>
							<p className="font-semibold text-lg">{capitalize(me.username.split("@")[0])}</p>
						</DropdownItem>
						<DropdownItem key="logout" color="danger" onClick={async () => {
							if (typeof window !== "undefined") {
								if (window.confirm("do you really want to signout?")) {
									await signout();
									router.replace("/login");
								}
							}
						}}>
							Log Out
						</DropdownItem>
						<DropdownItem key="logout" color="danger" onClick={async () => {
							if (typeof window !== "undefined") {
								if (window.confirm("do you really want to restart server?")) {
									await restart();
								}
							}
						}}>
							Restart Server
						</DropdownItem>
					</DropdownMenu>
				</Dropdown>
			</NavbarContent>

			<NavbarContent className="sm:hidden basis-1 pl-4" justify="end">
				<Link isExternal href={siteConfig.links.github} aria-label="Github">
					<GithubIcon className="text-default-500" />
				</Link>
				{serversMenu}
				{gamesMenu}
				<ThemeSwitch />
				<NavbarMenuToggle />
			</NavbarContent>

			<NavbarMenu>
				{searchInput}
				<div className="mx-4 mt-2 flex flex-col gap-2">
					{items.map((item, index) => (
						<NavbarMenuItem key={`${routeKey}-${item.key}-${index}`}>
							<Link
								color={
									index === 2
										? "primary"
										: index === siteConfig.navMenuItems.length - 1
											? "danger"
											: "foreground"
								}
								href="#"
								size="lg"
							>
								{item.title}
							</Link>
						</NavbarMenuItem>
					))}
				</div>
			</NavbarMenu>
		</NextUINavbar>
	);
};
