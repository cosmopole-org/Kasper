
import { ReactNode } from "react"

export default function Grid({ xs, item, container, children, style, className }: Readonly<{ style?: any, className?: string, xs?: number, item?: boolean, container?: boolean, children?: ReactNode }>) {
    if (container) {
        return (
            <div className={`grid grid-cols-${xs ?? 4} ` + (className ?? "")} style={style}>
                {children}
            </div>
        )
    } else if (item) {
        return (
            <div style={style}>
                {children}
            </div>
        )
    }
}