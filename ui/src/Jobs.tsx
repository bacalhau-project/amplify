import { Datagrid, List, TextField } from "react-admin";


export const JobList = () => (
    <List>
        <Datagrid rowClick="show">
            <TextField source="id" sortable={false} />
            <TextField source="attributes.image" sortable={false} />
            <TextField source="attributes.entrypoint" sortable={false} />
        </Datagrid>
    </List>
);