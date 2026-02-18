import type { ColumnDef, Table as TanstackTable } from '@tanstack/react-table'

export interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[]
  data: TData[]
  searchKey?: string
  searchPlaceholder?: string
  className?: string
}

export interface DataTablePaginationProps<TData> {
  table: TanstackTable<TData>
}

export interface DataTableColumnHeaderProps<TData, TValue>
  extends React.HTMLAttributes<HTMLDivElement> {
  column: import('@tanstack/react-table').Column<TData, TValue>
  title: string
}

export interface DataTableToolbarProps<TData> {
  table: TanstackTable<TData>
  searchKey?: string
  searchPlaceholder?: string
}
