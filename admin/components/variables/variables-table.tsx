"use client"

import React, { useEffect, useState } from "react";
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
    Chip,
    Pagination,
    Selection,
    ChipProps,
    SortDescriptor
} from "@nextui-org/react";
import { SearchIcon, ChevronDownIcon, PlusIcon, VerticalDotsIcon } from "../../components/icons";
import { columns, dataTypeOptions } from "../../api/offline/constants";
import { capitalize } from "../../utils";
import { switchVariablesEditModal } from "@/components/variables/edit-modals";
import Icon from "../elements/icon";
import { hookstate, useHookstate } from "@hookstate/core";
import { Variable } from "@/types";
import { readAllMetadata, readMetadata, updateMetadata } from "@/api/online/metadata";

const dataTypeColorMap: Record<string, ChipProps["color"]> = {
    object: "success",
    array: "danger",
    number: "warning",
    text: "primary",
};

const INITIAL_VISIBLE_COLUMNS = ["key", "type", "dataType", "value", "actions"];

const variablesListTrigger = hookstate("");
export const updateVariablesList = () => variablesListTrigger.set(Math.random().toString());
export const variablesHolder = hookstate<{ [key: string]: any }[]>([]);
export const updateVariablesData = (v: { [key: string]: any }[]) => {
    variablesHolder.set(v);
}
export const variablesTag = hookstate("");

export default function VariablesTable({ gameKey = "cars", mode = "prod" }: { gameKey: string, mode: string }) {
    const game = gameKey;
    const variablesState = useHookstate(variablesHolder);
    const tag = useHookstate(variablesTag).get({ noproxy: true });
    let variables: { [key: string]: any } = {};
    if (variablesState.get({ noproxy: true })) {
        let temp = variablesState.get({ noproxy: true });
        for (let i = 0; i < temp.length; i++) {
            if (temp[i].id === tag) {
                variables = temp[i].data;
            }
        }
    }
    const variablesTriggerState = useHookstate(variablesListTrigger);
    const [filterValue, setFilterValue] = React.useState("");
    const [selectedKeys, setSelectedKeys] = React.useState<Selection>(new Set([]));
    const [visibleColumns, setVisibleColumns] = React.useState<Selection>(new Set(INITIAL_VISIBLE_COLUMNS));
    const [dataTypeFilter, setDataTypeFilter] = React.useState<Selection>("all");
    const [rowsPerPage, setRowsPerPage] = React.useState(50);
    const [sortDescriptor, setSortDescriptor] = React.useState<SortDescriptor>({
        column: "age",
        direction: "ascending",
    });

    useEffect(() => {
        readAllMetadata(game).then(data => {
            updateVariablesData(data);
            variablesTag.set(data[0].id);
        });
    }, [gameKey, mode]);

    const [page, setPage] = React.useState(1);

    const hasSearchFilter = Boolean(filterValue);

    const headerColumns = React.useMemo(() => {
        if (visibleColumns === "all") return columns;

        return columns.filter((column) => Array.from(visibleColumns).includes(column.uid));
    }, [visibleColumns]);

    const filteredItems = React.useMemo(() => {
        if (variables) {
            let vars: Variable[] = []
            Object.keys(variables).forEach(key => {
                if (variables[key] !== null) {
                    let t: string = typeof variables[key];
                    if (Array.isArray(variables[key])) {
                        t = "array";
                    }
                    vars.push({ key, value: variables[key], type: t === "string" ? "text" : t });
                }
            })
            let filteredVariables = [...vars];

            if (hasSearchFilter) {
                filteredVariables = filteredVariables.filter((variable) =>
                    variable.key.toLowerCase().includes(filterValue.toLowerCase()),
                );
            }
            if (dataTypeFilter !== "all" && Array.from(dataTypeFilter).length !== dataTypeOptions.length) {
                filteredVariables = filteredVariables.filter((variable) => {
                    Array.from(dataTypeFilter).includes(variable.type)
                });
            }

            return filteredVariables;
        }
        return [];
    }, [variablesTriggerState.get({ noproxy: true }), variables, filterValue, dataTypeFilter]);

    const pages = Math.ceil(filteredItems.length / rowsPerPage);

    const items = React.useMemo(() => {
        const start = (page - 1) * rowsPerPage;
        const end = start + rowsPerPage;

        return filteredItems.slice(start, end);
    }, [page, filteredItems, rowsPerPage]);

    const sortedItems = React.useMemo(() => {
        return [...items].sort((a: Variable, b: Variable) => {
            const first = a[sortDescriptor.column as keyof Variable] as number;
            const second = b[sortDescriptor.column as keyof Variable] as number;
            let cmp = 0;
            if (first < second) {
                cmp = -1;
            } else if (first > second) {
                cmp = 1;
            }
            return sortDescriptor.direction === "descending" ? -cmp : cmp;
        });
    }, [sortDescriptor, items]);

    const renderCell = React.useCallback((variable: Variable, columnKey: React.Key) => {
        const cellValue = variable[columnKey as keyof Variable];

        switch (columnKey) {
            case "key":
                return (
                    <div className="flex flex-col">
                        <p className="text-bold text-small">{cellValue}</p>
                    </div>
                );
            case "type":
                return (
                    <div>
                        <Icon name="variables" size={[20, 20]} className="ml-3" />
                        <div className="text-bold text-tiny capitalize text-default-400 mt-2">variable</div>
                    </div>
                );
            case "dataType":
                return (
                    <Chip className="capitalize" color={dataTypeColorMap[variable.type]} size="sm" variant="flat">
                        {variable.type}
                    </Chip>
                );
            case "value": {
                let value = "";
                if (variable.type === 'number') {
                    value = cellValue.toString();
                } else if (variable.type === 'array' || variable.type === 'object') {
                    value = JSON.stringify(cellValue);
                } else if (variable.type === 'boolean') {
                    value = cellValue.toString();
                } else {
                    value = cellValue;
                }
                return (
                    <div className="flex flex-col">
                        <p className="text-bold text-small">{value}</p>
                    </div>
                );
            }
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
                                    switchVariablesEditModal(true, { key: variable.key });
                                } else if (key === 'delete') {
                                    if (typeof window !== 'undefined') {
                                        if (window.confirm('do you want to delete [' + variable.key + ']')) {
                                            let data = [ ...variablesState.get({ noproxy: true }) ];
                                            variablesState.set(data);
                                            await updateMetadata(tag, { [variable.key]: null });
                                            updateVariablesData(await readAllMetadata(game));
                                            updateVariablesList();
                                        }
                                    }
                                }
                            }}>
                                <DropdownItem key={'edit'}>Edit</DropdownItem>
                                <DropdownItem key={'delete'}>Delete</DropdownItem>
                            </DropdownMenu>
                        </Dropdown>
                    </div>
                );
            default:
                return cellValue;
        }
    }, [tag]);

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
                        <Dropdown backdrop="blur">
                            <DropdownTrigger>
                                <Button
                                    variant="bordered"
                                    className="capitalize"
                                    endContent={<Icon name="dropdown" size={[12, 12]} />}
                                >
                                    <p className="mr-2">{tag}</p>
                                </Button>
                            </DropdownTrigger>
                            <DropdownMenu
                                aria-label="Single selection example"
                                variant="flat"
                                disallowEmptySelection
                                selectionMode="single"
                                selectedKeys={tag}
                                onAction={(key) => {
                                    variablesTag.set(key.toString());
                                }}
                            >
                                {
                                    variablesState.get({ noproxy: true }).map((item: any) => <DropdownItem key={item.id}>{item.id}</DropdownItem>)
                                }
                            </DropdownMenu>
                        </Dropdown>
                        <Dropdown>
                            <DropdownTrigger className="hidden sm:flex">
                                <Button endContent={<ChevronDownIcon className="text-small" />} variant="flat">
                                    Data Type
                                </Button>
                            </DropdownTrigger>
                            <DropdownMenu
                                disallowEmptySelection
                                aria-label="Table Columns"
                                closeOnSelect={false}
                                selectedKeys={dataTypeFilter}
                                selectionMode="multiple"
                                onSelectionChange={setDataTypeFilter}
                            >
                                {dataTypeOptions.map((dataType) => (
                                    <DropdownItem key={dataType.uid} className="capitalize">
                                        {capitalize(dataType.name)}
                                    </DropdownItem>
                                ))}
                            </DropdownMenu>
                        </Dropdown>
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
                                {columns.map((column) => (
                                    <DropdownItem key={column.uid} className="capitalize">
                                        {capitalize(column.name)}
                                    </DropdownItem>
                                ))}
                            </DropdownMenu>
                        </Dropdown>
                        <Button color="primary" endContent={<PlusIcon />} onClick={() => {
                            switchVariablesEditModal(true, { key: "-1" });
                        }}>
                            Add New
                        </Button>
                    </div>
                </div>
                <div className="flex justify-between items-center">
                    <span className="text-default-400 text-small">Total {sortedItems.length} variables</span>
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
        dataTypeFilter,
        visibleColumns,
        onSearchChange,
        onRowsPerPageChange,
        sortedItems.length,
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

    console.log(sortedItems)

    return (
        <div key={game} className="w-[calc(100% - 64px)] m-8">
            <Table
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
                <TableBody emptyContent={"No variables found"} items={sortedItems}>
                    {
                        sortedItems.map((item) => (
                            <TableRow key={item.key}>
                                {(columnKey) => <TableCell>{renderCell(item, columnKey)}</TableCell>}
                            </TableRow>
                        ))
                    }
                </TableBody>
            </Table>
        </div>
    );
}
