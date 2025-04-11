"use client"

import React, { useEffect } from "react";
import {
    Table,
    TableHeader,
    TableColumn,
    TableBody,
    TableRow,
    TableCell,
    Input,
    Button,
    DropdownTrigger,
    Dropdown,
    DropdownMenu,
    DropdownItem,
    Pagination,
    Selection,
    SortDescriptor,
    Chip
} from "@nextui-org/react";
import { SearchIcon, ChevronDownIcon, VerticalDotsIcon } from "../icons";
import { playerDataCols } from "../../api/offline/constants";
import { capitalize } from "../../utils";
import { hookstate, useHookstate } from "@hookstate/core";
import { Player } from "@/types";
import { readPlayersList } from "@/api/online/players";
import { switchPlayersEditModal } from "./edit-modals";
import { selectedGame } from "../navbar";

const INITIAL_VISIBLE_COLUMNS = ["id", "email", "playerName", "coin", "gem", "energy", "actions"];

const playersListTrigger = hookstate("");
export const updatePlayersList = () => playersListTrigger.set(Math.random().toString());
export const playersHolder = hookstate<Player[]>([]);
export const updatePlayersListsData = (v: Player[]) => {
    playersHolder.set(v);
}

export default function PlayersTable({ gameKey = 'cars', mode = 'prod' }: Readonly<{ gameKey: string, mode: string }>) {
    const game = gameKey;
    const playersState = useHookstate(playersHolder);
    const players = playersState.get({ noproxy: true })
    const playersTriggerState = useHookstate(playersListTrigger);
    const [filterValue, setFilterValue] = React.useState("");
    const [selectedKeys, setSelectedKeys] = React.useState<Selection>(new Set([]));
    const [visibleColumns, setVisibleColumns] = React.useState<Selection>(new Set(INITIAL_VISIBLE_COLUMNS));
    const [rowsPerPage, setRowsPerPage] = React.useState(50);
    const [sortDescriptor, setSortDescriptor] = React.useState<SortDescriptor>({
        column: "age",
        direction: "ascending",
    });

    const [page, setPage] = React.useState(1);
    const [totalCount, setTotalCount] = React.useState(0);

    useEffect(() => {
        let offset = (page - 1) * rowsPerPage;
        let count = rowsPerPage
        readPlayersList(game, offset, count, filterValue.length > 0 ? filterValue : undefined).then((result: any) => {
            updatePlayersListsData(result[0]);
            setTotalCount(result[1]);
        });
    }, [game, mode, page, filterValue, playersTriggerState.get({ noproxy: true })]);

    const hasSearchFilter = Boolean(filterValue);

    const headerColumns = React.useMemo(() => {
        if (visibleColumns === "all") return playerDataCols;

        return playerDataCols.filter((column) => Array.from(visibleColumns).includes(column.uid));
    }, [visibleColumns]);

    const filteredItems = React.useMemo(() => {
        if (players && Array.isArray(players)) {
            let filteredPlayers = [...players]

            if (hasSearchFilter) {
                filteredPlayers = filteredPlayers.filter((player: Player) =>
                    player.email.toLowerCase().includes(filterValue.toLowerCase()) ||
                    player.playerName.toLowerCase().includes(filterValue.toLowerCase()) ||
                    (player.id === filterValue)
                );
            }

            return filteredPlayers;
        }
        return [];
    }, [playersTriggerState.get({ noproxy: true }), players, filterValue]);

    const pages = Math.ceil(totalCount / rowsPerPage);

    const items = filteredItems;

    const sortedItems = React.useMemo(() => {
        return [...items].sort((a: Player, b: Player) => {
            const first = a[sortDescriptor.column as keyof Player] as number;
            const second = b[sortDescriptor.column as keyof Player] as number;
            let cmp = 0;
            if (first < second) {
                cmp = -1;
            } else if (first > second) {
                cmp = 1;
            }
            return sortDescriptor.direction === "descending" ? -cmp : cmp;
        });
    }, [sortDescriptor, items]);

    const renderCell = React.useCallback((player: Player, columnKey: React.Key) => {
        const cellValue = player[columnKey as keyof Player];

        switch (columnKey) {
            case "email":
                return (
                    <Chip className="capitalize" color={"success"} size="sm" variant="flat">
                        {cellValue}
                    </Chip>
                );
            case "actions":
                return (
                    <div className="relative flex justify-end items-center gap-2">
                        <Dropdown>
                            <DropdownTrigger>
                                <Button isIconOnly size="sm" variant="light" className="mr-auto ml-0">
                                    <VerticalDotsIcon className="text-default-300" />
                                </Button>
                            </DropdownTrigger>
                            <DropdownMenu onAction={async (key) => {
                                if (key === 'edit') {
                                    switchPlayersEditModal(true, { id: player.id });
                                }
                            }}>
                                <DropdownItem key={'edit'}>Edit</DropdownItem>
                            </DropdownMenu>
                        </Dropdown>
                    </div>
                );
            default:
                return cellValue;
        }
    }, []);

    const onNextPage = React.useCallback(() => {
        if (page < pages) {
            setPage(page + 1);
        }
    }, [page, pages]);

    const onPreviousPage = React.useCallback(() => {
        if (page > 1) {
            setPage(page - 1);
        }
    }, [page]);

    const onRowsPerPageChange = React.useCallback((e: React.ChangeEvent<HTMLSelectElement>) => {
        setRowsPerPage(Number(e.target.value));
        setPage(1);
    }, []);

    const onSearchChange = React.useCallback((value?: string) => {
        if (value) {
            setFilterValue(value);
            setPage(1);
        } else {
            setFilterValue("");
        }
    }, []);

    const onClear = React.useCallback(() => {
        setFilterValue("")
        setPage(1)
    }, [])

    const topContent = React.useMemo(() => {
        return (
            <div className="flex flex-col gap-4">
                <div className="flex justify-between gap-3 items-end">
                    <Input
                        isClearable
                        className="w-full sm:max-w-[44%]"
                        placeholder="Search by name..."
                        startContent={<SearchIcon />}
                        value={filterValue}
                        onClear={() => onClear()}
                        onValueChange={onSearchChange}
                    />
                    <div className="flex gap-3">
                        <Dropdown>
                            <DropdownTrigger className="hidden sm:flex">
                                <Button endContent={<ChevronDownIcon className="text-small" />} variant="flat">
                                    Columns
                                </Button>
                            </DropdownTrigger>
                            <DropdownMenu
                                disallowEmptySelection
                                aria-label="Table Columns"
                                closeOnSelect={false}
                                selectedKeys={visibleColumns}
                                selectionMode="multiple"
                                onSelectionChange={setVisibleColumns}
                            >
                                {playerDataCols.map((column) => (
                                    <DropdownItem key={column.uid} className="capitalize">
                                        {capitalize(column.name)}
                                    </DropdownItem>
                                ))}
                            </DropdownMenu>
                        </Dropdown>
                    </div>
                </div>
                <div className="flex justify-between items-center">
                    <span className="text-default-400 text-small">Total {totalCount} players</span>
                    <label className="flex items-center text-default-400 text-small">
                        Rows per page: <select
                            className="bg-transparent outline-none text-default-400 text-small"
                            onChange={onRowsPerPageChange}
                        >
                            <option value="50">50</option>
                            <option value="100">100</option>
                            <option value="200">200</option>
                        </select>
                    </label>
                </div>
            </div>
        );
    }, [
        filterValue,
        visibleColumns,
        onSearchChange,
        onRowsPerPageChange,
        totalCount,
        hasSearchFilter,
    ]);

    const bottomContent = React.useMemo(() => {
        return (
            <div className="py-2 px-2 flex justify-between items-center">
                <span className="w-[30%] text-small text-default-400">
                    {selectedKeys === "all"
                        ? "All items selected"
                        : `${selectedKeys.size} of ${filteredItems.length} selected`}
                </span>
                <Pagination
                    isCompact
                    showControls
                    showShadow
                    color="primary"
                    page={page}
                    total={pages}
                    onChange={setPage}
                />
                <div className="hidden sm:flex w-[30%] justify-end gap-2">
                    <Button isDisabled={pages === 1} size="sm" variant="flat" onPress={onPreviousPage}>
                        Previous
                    </Button>
                    <Button isDisabled={pages === 1} size="sm" variant="flat" onPress={onNextPage}>
                        Next
                    </Button>
                </div>
            </div>
        );
    }, [selectedKeys, items.length, page, pages, hasSearchFilter]);

    return (
        <div className={`m-8`} style={{ maxWidth: '100%' }}>
            <Table
                classNames={{
                    wrapper: "w-[calc(100vw-212px)]"
                }}
                isHeaderSticky
                bottomContent={bottomContent}
                bottomContentPlacement="outside"
                selectionMode="none"
                sortDescriptor={sortDescriptor}
                topContent={topContent}
                topContentPlacement="outside"
                onSelectionChange={setSelectedKeys}
                onSortChange={setSortDescriptor}
            >
                <TableHeader columns={headerColumns}>
                    {(column) => (
                        <TableColumn
                            key={column.uid}
                            align={column.uid === "actions" ? "center" : "start"}
                            allowsSorting={column.sortable}
                        >
                            {column.name}
                        </TableColumn>
                    )}
                </TableHeader>
                <TableBody emptyContent={"No player found"} items={sortedItems}>
                    {
                        sortedItems.map((item) => (
                            <TableRow key={item.id}>
                                {(columnKey) => <TableCell>{renderCell(item, columnKey)}</TableCell>}
                            </TableRow>
                        ))
                    }
                </TableBody>
            </Table>
        </div>
    );
}
