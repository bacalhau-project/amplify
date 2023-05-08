import { Datagrid, List, NumberField, TextField } from "react-admin";

export const ContentList = () => (
    <List bulkActionButtons={false} actions={false} title={<div></div>} sort={{ field: 'meta.count', order: 'DESC' }}>
        <Datagrid rowClick={false} bulkActionButtons={false} >
            <TextField source="id" label="Content-Type" sortable={false} />
            <NumberField source="meta.count" label="Count" />
        </Datagrid>
    </List>
);

export const RecentResultList = () => (
    <List bulkActionButtons={false} actions={false} title={<div></div>} sort={{ field: 'meta.created_at', order: 'DESC' }}>
        <Datagrid rowClick={false} bulkActionButtons={false} >
            <NumberField source="meta.created_at" noWrap sortable={false} />
            <TextField source="id" sortable={false} />
        </Datagrid>
    </List>
);
