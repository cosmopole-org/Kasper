import { Avatar, Card } from "@nextui-org/react";
import HomeRoomsList from "@/components/home/home-rooms-list";
import Icon from "@/components/elements/icon";
import HomeInboxModal from "@/components/home/home-inbox-modal";
import { Actions, useTheme } from "@/api/client/states";
import { getUsers } from "@/api/client/constants";
import { useEffect, useState } from "react";
import { api } from "@/index";
import { Point, User } from "@/api/sigma/models";

export default function HomePage() {
	const [selectedSpaceId, setSelectedSpaceId] = useState("");
	const [spaces, setSpaces] = useState<Point[]>([]);
	const [interactsDict, setInteractsDict] = useState<{ [id: string]: Point }>({});
	const [spacesDict, setSpacesDict] = useState<{ [id: string]: Point }>({});
	const { navAsHome } = useTheme();
	useEffect(() => {
		const interactsObservable = api.store?.db.points.find().$;
		let interactsSub = interactsObservable?.subscribe(inters => {
			let dict: { [id: string]: Point } = {};
			let usersDict: { [id: string]: User } = {};
			inters.forEach(inter => {
				dict[inter.id] = inter as Point;
			});
			api.store?.db.users.find({
				selector: {
					id: {
						$in: inters.map(inter => inter.peerId)
					}
				}
			}).exec().then((users) => {
				users.forEach((user) => {
					usersDict[user.id] = user;
				});
				Promise.all(inters.map(async inter => {
					if (inter.peerId) {
						if (!usersDict[inter.peerId]) {
							let user = await api.services?.users.get(inter.peerId);
							if (user) {
								usersDict[user.id] = user;
							}
						}
						(inter as any).participant = usersDict[inter.peerId];
					}
				})).then(() => {
					setInteractsDict(dict);
				})
			})
		});
		const spacesObservable = api.store?.db.points.find().$;
		let spacesSub = spacesObservable?.subscribe(ss => {
			let dict: { [id: string]: Point } = {};
			ss.forEach(s => { dict[s.id] = s as Point; })
			setSpacesDict(dict);
			if (selectedSpaceId === "") {
				setSelectedSpaceId(ss.length > 0 ? (ss[0] as any).id : "");
			}
			console.log(dict);
			setSpaces(ss as Point[]);
		});
		if (navAsHome === "true") {
			Actions.updateHomeMenuState(true);
		}
		return () => {
			interactsSub?.unsubscribe();
			spacesSub?.unsubscribe();
		}
	}, []);
	const homeSpace = spaces[0];
	console.log("test: ", spacesDict[selectedSpaceId]?.peerId ? ((spacesDict[selectedSpaceId] as any)?.participant as User)?.name : spacesDict[selectedSpaceId]?.title);
	return (
		<div className="relative flex flex-col h-screen w-full bg-s-white dark:bg-content2">
			<Card shadow={navAsHome ? 'none' : 'md'} className="w-20 h-full bg-s-white dark:bg-content2 pl-2 pt-12 fixed overflow-y-auto" style={{ borderRadius: 0 }}>
				{homeSpace ? (
					<div
						onClick={() => {
							setSelectedSpaceId(spaces[0].id);
						}}
						key={homeSpace.id} className="w-12 h-12 mt-6 ml-2" style={{ borderRadius: '50%' }}>
						<Card className="w-full h-full bg-content3 pt-2 pl-2 pr-2" shadow="md" style={{ borderRadius: '50%', minHeight: 48 }}>
							<Icon name="home" size={[32, 32]} />
						</Card>
					</div>
				) : null}
				<div
					onClick={() => {
						Actions.openCreateSpaceModal();
					}}
					key={"add"} className="w-12 h-12 mt-6 ml-2" style={{ borderRadius: '50%' }}>
					<Card className="w-full h-full bg-content3 pt-2 pl-2 pr-2" shadow="md" style={{ borderRadius: '50%', minHeight: 48 }}>
						<Icon name="add" size={[32, 32]} />
					</Card>
				</div>
				{
					spaces.slice(1).map((item: any) => {
						if (interactsDict[item.spaceId]) {
							let participant = (interactsDict[item.spaceId] as any).participant as User;
							return (
								<div
									onClick={() => {
										setSelectedSpaceId(item.id);
									}}
									key={item.id} className="w-12 h-12 bg-white dark: bg-content1 mt-6 ml-2" style={{ borderRadius: '50%' }}>
									<Avatar alt={participant.name} className="w-full h-full" size="lg" src={participant.avatar} isBordered />
								</div>
							);
						} else {
							return (
								<div
									onClick={() => {
										setSelectedSpaceId(item.id);
									}}
									key={item.id} className="w-12 h-12 bg-white dark: bg-content1 mt-6 ml-2" style={{ borderRadius: '50%' }}>
									<Avatar alt={item.title} className="w-full h-full" size="lg" src={getUsers()[item.avatar]?.avatar} isBordered />
								</div>
							);
						}
					})
				}
			</Card>
			<div className={`fixed left-20 h-full overflow-hidden`} style={{ width: navAsHome ? '100%' : 'calc(100% - 144px)', top: navAsHome === "true" ? 58 : 0, borderRadius: navAsHome ? '32px 0px 0px 0px' : 0 }}>
				<HomeRoomsList title={spacesDict[selectedSpaceId]?.peerId ? ((interactsDict[selectedSpaceId] as any)?.participant as User)?.name : spacesDict[selectedSpaceId]?.title} isNavAsHome={navAsHome === "true"} pointId={selectedSpaceId} />
			</div>
			<HomeInboxModal />
		</div>
	);
}
